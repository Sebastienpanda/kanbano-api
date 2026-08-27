package handler

import (
	"log"
	"net/http"

	"kanbano-api/internal/media"
	"kanbano-api/internal/models"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/storage"
	"kanbano-api/internal/ws"

	"github.com/google/uuid"
)

const maxAvatarUpload = 5 << 20 // 5 MiB

type UserHandler struct {
	repo  *repository.UserRepository
	store *storage.Client
	hub   *ws.Hub
}

type updateMeBody struct {
	Name string `json:"name"`
}

// meResponse is the wire shape for /me: the user, plus the avatar URLs when set.
type meResponse struct {
	models.User
	Avatar *models.AvatarSet `json:"avatar"`
}

func NewUserHandler(repo *repository.UserRepository, store *storage.Client, hub *ws.Hub) *UserHandler {
	return &UserHandler{repo: repo, store: store, hub: hub}
}

func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	user, err := h.repo.GetByID(r.Context(), userID)
	if handleRepoError(w, err, "user not found") {
		return
	}

	writeJSON(w, http.StatusOK, h.response(user))
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	body, ok := decodeJSON[updateMeBody](w, r)
	if !ok {
		return
	}
	if body.Name == "" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.repo.UpdateName(r.Context(), userID, body.Name)
	if handleRepoError(w, err, "user not found") {
		return
	}

	h.broadcastAndWrite(w, userID, user, ws.UserUpdated)
}

// UploadAvatar accepts a multipart "file" field, generates every avatar
// derivative, stores them in object storage and points the user at the new version.
func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		http.Error(w, "avatar storage unavailable", http.StatusServiceUnavailable)
		return
	}

	userID := userIDFromContext(r)

	current, err := h.repo.GetByID(r.Context(), userID)
	if handleRepoError(w, err, "user not found") {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarUpload)
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	derivatives, err := media.AvatarDerivatives(file)
	if err != nil {
		http.Error(w, "invalid image", http.StatusBadRequest)
		return
	}

	version, err := media.NewVersion()
	if err != nil {
		serverError(w, err)
		return
	}

	for _, d := range derivatives {
		key := media.AvatarObjectKey(userID, version, d.Format, d.Size)
		if err := h.store.Put(r.Context(), key, d.Data, d.ContentType); err != nil {
			_ = h.store.RemovePrefix(r.Context(), media.AvatarVersionPrefix(userID, version))
			serverError(w, err)
			return
		}
	}

	user, err := h.repo.SetAvatar(r.Context(), userID, version)
	if handleRepoError(w, err, "user not found") {
		_ = h.store.RemovePrefix(r.Context(), media.AvatarVersionPrefix(userID, version))
		return
	}

	if old := current.AvatarVersion; old != nil && *old != "" && *old != version {
		if err := h.store.RemovePrefix(r.Context(), media.AvatarVersionPrefix(userID, *old)); err != nil {
			log.Printf("avatar: purge old version %s: %v", *old, err)
		}
	}

	h.broadcastAndWrite(w, userID, user, ws.AvatarUpdated)
}

// DeleteAvatar removes every stored avatar object for the user and clears the
// pointer.
func (h *UserHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		http.Error(w, "avatar storage unavailable", http.StatusServiceUnavailable)
		return
	}

	userID := userIDFromContext(r)

	if err := h.store.RemovePrefix(r.Context(), media.AvatarUserPrefix(userID)); err != nil {
		serverError(w, err)
		return
	}

	user, err := h.repo.ClearAvatar(r.Context(), userID)
	if handleRepoError(w, err, "user not found") {
		return
	}

	h.broadcastAndWrite(w, userID, user, ws.AvatarDeleted)
}

func (h *UserHandler) broadcastAndWrite(w http.ResponseWriter, userID uuid.UUID, user models.User, event ws.EventType) {
	resp := h.response(user)
	h.hub.Broadcast(userID, ws.Event{Type: event, Data: resp})
	writeJSON(w, http.StatusOK, resp)
}

func (h *UserHandler) response(user models.User) meResponse {
	return meResponse{User: user, Avatar: h.avatarSet(user)}
}

func (h *UserHandler) avatarSet(user models.User) *models.AvatarSet {
	if h.store == nil || user.AvatarVersion == nil || *user.AvatarVersion == "" {
		return nil
	}
	version := *user.AvatarVersion

	formats := func(size int) models.AvatarFormats {
		return models.AvatarFormats{
			Avif: h.store.URL(media.AvatarObjectKey(user.ID, version, media.FormatAVIF, size)),
			Webp: h.store.URL(media.AvatarObjectKey(user.ID, version, media.FormatWebP, size)),
			Png:  h.store.URL(media.AvatarObjectKey(user.ID, version, media.FormatPNG, size)),
		}
	}

	return &models.AvatarSet{Size45: formats(45), Size100: formats(100)}
}
