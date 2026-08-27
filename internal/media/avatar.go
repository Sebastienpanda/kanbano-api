package media

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	"image/png"
	"io"

	_ "image/jpeg" // register JPEG decoder for image.Decode

	"github.com/gen2brain/avif"
	"github.com/gen2brain/webp"
	"github.com/google/uuid"
	"golang.org/x/image/draw"
)

const (
	FormatAVIF = "avif"
	FormatWebP = "webp"
	FormatPNG  = "png"
)

var (
	avatarSizes   = []int{45, 100}
	avatarFormats = []string{FormatAVIF, FormatWebP, FormatPNG}
)

type Derivative struct {
	Size        int
	Format      string
	ContentType string
	Data        []byte
}

func AvatarDerivatives(r io.Reader) ([]Derivative, error) {
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode avatar: %w", err)
	}

	sr := squareBounds(src.Bounds())
	out := make([]Derivative, 0, len(avatarSizes)*len(avatarFormats))

	for _, size := range avatarSizes {
		dst := image.NewRGBA(image.Rect(0, 0, size, size))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, sr, draw.Src, nil)

		for _, format := range avatarFormats {
			data, contentType, err := encode(dst, format)
			if err != nil {
				return nil, err
			}
			out = append(out, Derivative{
				Size:        size,
				Format:      format,
				ContentType: contentType,
				Data:        data,
			})
		}
	}

	return out, nil
}

func NewVersion() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("avatar version: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

func AvatarUserPrefix(userID uuid.UUID) string {
	return "avatars/" + userID.String() + "/"
}

func AvatarVersionPrefix(userID uuid.UUID, version string) string {
	return AvatarUserPrefix(userID) + version + "/"
}

func AvatarObjectKey(userID uuid.UUID, version, format string, size int) string {
	return fmt.Sprintf("%s%s/%d.%s", AvatarVersionPrefix(userID, version), format, size, format)
}

func squareBounds(b image.Rectangle) image.Rectangle {
	side := b.Dx()
	if b.Dy() < side {
		side = b.Dy()
	}
	x0 := b.Min.X + (b.Dx()-side)/2
	y0 := b.Min.Y + (b.Dy()-side)/2
	return image.Rect(x0, y0, x0+side, y0+side)
}

func encode(img image.Image, format string) (data []byte, contentType string, err error) {
	var buf bytes.Buffer
	switch format {
	case FormatPNG:
		if err = png.Encode(&buf, img); err != nil {
			return nil, "", fmt.Errorf("encode png: %w", err)
		}
		return buf.Bytes(), "image/png", nil
	case FormatWebP:
		if err = webp.Encode(&buf, img, webp.Options{Quality: 90, Method: 6}); err != nil {
			return nil, "", fmt.Errorf("encode webp: %w", err)
		}
		return buf.Bytes(), "image/webp", nil
	case FormatAVIF:
		if err = avif.Encode(&buf, img, avif.Options{Quality: 60, Speed: 6}); err != nil {
			return nil, "", fmt.Errorf("encode avif: %w", err)
		}
		return buf.Bytes(), "image/avif", nil
	default:
		return nil, "", fmt.Errorf("unknown avatar format %q", format)
	}
}
