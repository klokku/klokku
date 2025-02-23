package user

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/klokku/klokku/internal/rest"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
	"time"
)

type UserDTO struct {
	Id          int         `json:"id"`
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

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Creating user")

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
	log.Debug("Creating new user: ", user)

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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Debug("Created user: ", createdUser)

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(userToDTO(&createdUser)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) CurrentUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Getting current user")

	currentUser, err := h.userService.GetCurrentUser(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(userToDTO(&currentUser)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

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

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Deleting user")

	vars := mux.Vars(r)
	userIdString := vars["id"]
	userId, err := strconv.ParseInt(userIdString, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Debug("Deleting user with id: ", userId)
	err = h.userService.DeleteUser(r.Context(), int(userId))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

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

func (h *Handler) GetPhoto(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/jpeg")
	log.Trace("Getting user photo")

	vars := mux.Vars(r)
	userIdString := vars["userId"]
	if userIdString != "" {
		userId, err := strconv.ParseInt(userIdString, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		photo, err := h.userService.GetUserPhoto(r.Context(), int(userId))
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
		Id:          user.Id,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Settings:    settingsToDTO(user.Settings),
	}
}

func settingsToDTO(settings Settings) SettingsDTO {
	return SettingsDTO{
		Timezone:          settings.Timezone,
		WeekStartDay:      settings.WeekFirstDay.String(),
		EventCalendarType: settings.EventCalendarType,
		GoogleCalendar: GoogleCalendarSettingsDTO{
			CalendarId: settings.GoogleCalendar.CalendarId,
		},
	}
}

func dtoToUser(userDTO UserDTO) User {
	return User{
		Id:          userDTO.Id,
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
