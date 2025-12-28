package user

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/klokku/klokku/internal/rest"
	log "github.com/sirupsen/logrus"
)

type UserDTO struct {
	Uid         string      `json:"uid"`
	Username    string      `json:"username"`
	DisplayName string      `json:"displayName"`
	Settings    SettingsDTO `json:"settings"`
}

type SettingsDTO struct {
	Timezone          string                    `json:"timezone"`
	WeekStartDay      string                    `json:"weekStartDay"`
	EventCalendarType EventCalendarType         `json:"eventCalendarType"`
	GoogleCalendar    GoogleCalendarSettingsDTO `json:"googleCalendar"`
}

type GoogleCalendarSettingsDTO struct {
	CalendarId string `json:"calendarId"`
}

type Handler struct {
	userService Service
}

func NewHandler(userService Service) *Handler {
	return &Handler{
		userService: userService,
	}
}

// CreateUser godoc
// @Summary Create a new user
// @Description Register a new user in the system
// @Tags User
// @Accept json
// @Produce json
// @Param user body UserDTO true "User"
// @Success 201 {object} UserDTO
// @Failure 400 {object} rest.ErrorResponse "Invalid request"
// @Failure 403 {string} string "User not found"
// @Router /api/user [post]
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Debug("Creating user")

	var user UserDTO
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Invalid request body format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
		}
		return
	}
	log.Tracef("Creating new user: %+v", user)

	if len(user.Username) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Username is required",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	if len(user.DisplayName) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Display name is required",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	createdUser, err := h.userService.CreateUser(r.Context(), dtoToUser(user))
	if err != nil {
		if errors.Is(err, ErrUserDataInvalid) {
			w.WriteHeader(http.StatusBadRequest)
			encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
				Error: "Invalid user data",
			})
			if encodeErr != nil {
				http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			}
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Tracef("Created user: %+v", createdUser)

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(userToDTO(&createdUser)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// CurrentUser godoc
// @Summary Get current user
// @Description Retrieve the currently authenticated user's information
// @Tags User
// @Produce json
// @Success 200 {object} UserDTO
// @Failure 403 {string} string "User not found"
// @Failure 404 {string} string "User Not Found"
// @Router /api/user/current [get]
// @Security XUserId
func (h *Handler) CurrentUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Getting current user")

	currentUser, err := h.userService.GetCurrentUser(r.Context())
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(userToDTO(&currentUser)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// UpdateUser godoc
// @Summary Update current user
// @Description Update the currently authenticated user's information
// @Tags User
// @Accept json
// @Produce json
// @Param user body UserDTO true "User"
// @Success 200 {object} UserDTO
// @Failure 400 {object} rest.ErrorResponse "Invalid request"
// @Failure 403 {string} string "User not found"
// @Router /api/user/current [put]
// @Security XUserId
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Updating user")

	var user UserDTO
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Invalid request body format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	log.Debug("Updating user: ", user)
	if len(user.Username) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Username is required",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	if len(user.DisplayName) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Display name is required",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
		}
	}

	updatedUser, err := h.userService.UpdateUser(r.Context(), dtoToUser(user))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Debug("Updated user: ", updatedUser)

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(userToDTO(&updatedUser)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// IsUsernameAvailable godoc
// @Summary Check username availability
// @Description Check if a username is available for registration
// @Tags User
// @Produce json
// @Param username query string true "Username to check"
// @Success 200 {object} object{available=bool}
// @Failure 400 {object} rest.ErrorResponse "Username is required"
// @Router /api/user/name-availability [get]
func (h *Handler) IsUsernameAvailable(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Checking if username is available")

	vars := mux.Vars(r)
	username := vars["username"]
	log.Debug("Checking availability of username: ", username)
	if len(username) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Username is required",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
		}
	}

	isAvailable, err := h.userService.IsUsernameAvailable(r.Context(), username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]bool{"available": isAvailable}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GetAvailableUsers godoc
// @Summary Get all users
// @Description Retrieve a list of all registered users
// @Tags User
// @Produce json
// @Success 200 {array} UserDTO
// @Failure 403 {string} string "User not found"
// @Router /api/user [get]
func (h *Handler) GetAvailableUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Getting available users")

	users, err := h.userService.GetAllUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	usersDTO := make([]UserDTO, 0, len(users))
	for _, user := range users {
		usersDTO = append(usersDTO, userToDTO(&user))
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(usersDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// DeleteUser godoc
// @Summary Delete a user
// @Description Delete a user by UID
// @Tags User
// @Param userUid path string true "User UID"
// @Success 204 "No Content"
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Router /api/user/{userUid} [delete]
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Deleting user")

	vars := mux.Vars(r)
	userUid := vars["userUid"]
	user, err := h.userService.GetUserByUid(r.Context(), userUid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Debug("Deleting user with id: ", user.Id)
	err = h.userService.DeleteUser(r.Context(), user.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UploadPhoto godoc
// @Summary Upload user photo
// @Description Upload a profile photo for the current user (max 3MB)
// @Tags User
// @Accept multipart/form-data
// @Param photo formData file true "User photo"
// @Success 200 "OK"
// @Failure 400 {object} rest.ErrorResponse "Image too large or invalid"
// @Failure 403 {string} string "User not found"
// @Router /api/user/current/photo [put]
// @Security XUserId
func (h *Handler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Uploading user photo")

	// Enforce a hard limit of 3MB on the request body
	r.Body = http.MaxBytesReader(w, r.Body, 3<<20) // 3MB hard limit on request body
	// arg 3 << 20 specifies a maximum upload of 3MB files
	err := r.ParseMultipartForm(3 << 20)
	if err != nil {
		log.Debugf("File is too large: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Image is too large",
			Details: "Maximum size is 3MB. Please try again with a smaller image.",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()
	log.Debugf("Uploaded File: %+v\n", header.Filename)
	log.Debugf("File Size: %+v\n", header.Size)
	log.Debugf("MIME Header: %+v\n", header.Header)

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.userService.StoreUserPhoto(r.Context(), fileBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetPhoto godoc
// @Summary Get user photo
// @Description Retrieve a user's profile photo. If userUid is provided, gets that user's photo, otherwise gets current user's photo
// @Tags User
// @Produce image/jpeg
// @Param userUid path string false "User UID (optional)"
// @Success 200 {file} image/jpeg
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Router /api/user/current/photo [get]
// @Router /api/user/{userUid}/photo [get]
func (h *Handler) GetPhoto(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/jpeg")
	log.Trace("Getting user photo")

	vars := mux.Vars(r)
	userUid := vars["userUid"]
	if userUid != "" {
		user, err := h.userService.GetUserByUid(r.Context(), userUid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		photo, err := h.userService.GetUserPhoto(r.Context(), user.Id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(photo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	photo, err := h.userService.GetCurrentUserPhoto(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(photo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DeletePhoto godoc
// @Summary Delete user photo
// @Description Remove the current user's profile photo
// @Tags User
// @Success 204 "No Content"
// @Failure 403 {string} string "User not found"
// @Router /api/user/current/photo [delete]
// @Security XUserId
func (h *Handler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Deleting user photo")

	err := h.userService.DeleteUserPhoto(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func userToDTO(user *User) UserDTO {
	return UserDTO{
		Uid:         user.Uid,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Settings:    settingsToDTO(user.Settings),
	}
}

func settingsToDTO(settings Settings) SettingsDTO {
	return SettingsDTO{
		Timezone:          settings.Timezone,
		WeekStartDay:      strings.ToLower(settings.WeekFirstDay.String()),
		EventCalendarType: settings.EventCalendarType,
		GoogleCalendar: GoogleCalendarSettingsDTO{
			CalendarId: settings.GoogleCalendar.CalendarId,
		},
	}
}

func dtoToUser(userDTO UserDTO) User {
	return User{
		Uid:         userDTO.Uid,
		Username:    userDTO.Username,
		DisplayName: userDTO.DisplayName,
		Settings:    dtoToSettings(userDTO.Settings),
	}
}

func dtoToSettings(settingsDTO SettingsDTO) Settings {
	return Settings{
		Timezone:          settingsDTO.Timezone,
		WeekFirstDay:      stringToWeekday(settingsDTO.WeekStartDay),
		EventCalendarType: settingsDTO.EventCalendarType,
		GoogleCalendar: GoogleCalendarSettings{
			CalendarId: settingsDTO.GoogleCalendar.CalendarId,
		},
	}
}

func stringToWeekday(day string) time.Weekday {
	switch day {
	case "monday":
		return time.Monday
	case "tuesday":
		return time.Tuesday
	case "wednesday":
		return time.Wednesday
	case "thursday":
		return time.Thursday
	case "friday":
		return time.Friday
	case "saturday":
		return time.Saturday
	case "sunday":
		return time.Sunday
	}
	return time.Monday
}
