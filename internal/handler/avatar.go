package handler

import (
	"kanbano-api/internal/media"
	"kanbano-api/internal/models"
	"kanbano-api/internal/storage"

	"github.com/google/uuid"
)

// avatarSet builds the public avatar URLs for a user's version, or nil when
// storage is unavailable or no avatar is set.
func avatarSet(store *storage.Client, userID uuid.UUID, version *string) *models.AvatarSet {
	if store == nil || version == nil || *version == "" {
		return nil
	}
	v := *version

	formats := func(size int) models.AvatarFormats {
		return models.AvatarFormats{
			Avif: store.URL(media.AvatarObjectKey(userID, v, media.FormatAVIF, size)),
			Webp: store.URL(media.AvatarObjectKey(userID, v, media.FormatWebP, size)),
			Png:  store.URL(media.AvatarObjectKey(userID, v, media.FormatPNG, size)),
		}
	}

	return &models.AvatarSet{Size45: formats(45), Size100: formats(100)}
}
