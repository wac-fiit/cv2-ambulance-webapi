package ambulance_wl

import (
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type implAmbulanceWaitingListAPI struct {
	logger zerolog.Logger
}

func NewAmbulanceWaitingListApi() AmbulanceWaitingListAPI {
	return &implAmbulanceWaitingListAPI{logger: log.With().Str("component", "ambulance-wl").Logger()}
}

func (o implAmbulanceWaitingListAPI) CreateWaitingListEntry(c *gin.Context) {
	updateAmbulanceFunc(c, func(c *gin.Context, ambulance *Ambulance) (*Ambulance, interface{}, int) {
		logger := o.logger.With().
			Str("method", "CreateWaitingListEntry").
			Str("ambulanceId", ambulance.Id).
			Str("ambulanceName", ambulance.Name).
			Logger()
		var entry WaitingListEntry

		if err := c.ShouldBindJSON(&entry); err != nil {
			logger.Error().Err(err).Msg("Failed to bind JSON")
			return nil, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Invalid request body",
				"error":   err.Error(),
			}, http.StatusBadRequest
		}

		if entry.PatientId == "" {
			logger.Error().Msg("Patient ID is required")
			logger.Trace().Msgf("Entry: %+v", entry)
			return nil, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Patient ID is required",
			}, http.StatusBadRequest
		}

		if entry.Id == "" || entry.Id == "@new" {
			logger.Debug().
				Str("entry-id", entry.Id).
				Msg("Generating new ID for entry")
			entry.Id = uuid.NewString()
		}

		conflictIndx := slices.IndexFunc(ambulance.WaitingList, func(waiting WaitingListEntry) bool {
			logger.Error().Msg("Entry already exists")
			return entry.Id == waiting.Id || entry.PatientId == waiting.PatientId
		})

		if conflictIndx >= 0 {
			return nil, gin.H{
				"status":  http.StatusConflict,
				"message": "Entry already exists",
			}, http.StatusConflict
		}

		ambulance.WaitingList = append(ambulance.WaitingList, entry)
		ambulance.reconcileWaitingList()
		// entry was copied by value return reconciled value from the list
		entryIndx := slices.IndexFunc(ambulance.WaitingList, func(waiting WaitingListEntry) bool {
			return entry.Id == waiting.Id
		})
		if entryIndx < 0 {
			logger.Error().Msg("Failed to find entry in waiting list after saving")
			return nil, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "Failed to save entry",
			}, http.StatusInternalServerError
		}
		logger.Info().
			Str("entry-id", ambulance.WaitingList[entryIndx].Id).
			Msg("Succesfully created patient entry")
		return ambulance, ambulance.WaitingList[entryIndx], http.StatusOK
	})
}

func (o implAmbulanceWaitingListAPI) DeleteWaitingListEntry(c *gin.Context) {
	updateAmbulanceFunc(c, func(c *gin.Context, ambulance *Ambulance) (*Ambulance, interface{}, int) {
		entryId := c.Param("entryId")

		if entryId == "" {
			return nil, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Entry ID is required",
			}, http.StatusBadRequest
		}

		entryIndx := slices.IndexFunc(ambulance.WaitingList, func(waiting WaitingListEntry) bool {
			return entryId == waiting.Id
		})

		if entryIndx < 0 {
			return nil, gin.H{
				"status":  http.StatusNotFound,
				"message": "Entry not found",
			}, http.StatusNotFound
		}

		ambulance.WaitingList = append(ambulance.WaitingList[:entryIndx], ambulance.WaitingList[entryIndx+1:]...)
		ambulance.reconcileWaitingList()
		return ambulance, nil, http.StatusNoContent
	})
}

func (o implAmbulanceWaitingListAPI) GetWaitingListEntries(c *gin.Context) {
	updateAmbulanceFunc(c, func(c *gin.Context, ambulance *Ambulance) (*Ambulance, interface{}, int) {
		result := ambulance.WaitingList
		if result == nil {
			result = []WaitingListEntry{}
		}
		// return nil ambulance - no need to update it in db
		return nil, result, http.StatusOK
	})
}

func (o implAmbulanceWaitingListAPI) GetWaitingListEntry(c *gin.Context) {
	updateAmbulanceFunc(c, func(c *gin.Context, ambulance *Ambulance) (*Ambulance, interface{}, int) {
		entryId := c.Param("entryId")

		if entryId == "" {
			return nil, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Entry ID is required",
			}, http.StatusBadRequest
		}

		entryIndx := slices.IndexFunc(ambulance.WaitingList, func(waiting WaitingListEntry) bool {
			return entryId == waiting.Id
		})

		if entryIndx < 0 {
			return nil, gin.H{
				"status":  http.StatusNotFound,
				"message": "Entry not found",
			}, http.StatusNotFound
		}

		// return nil ambulance - no need to update it in db
		return nil, ambulance.WaitingList[entryIndx], http.StatusOK
	})
}

func (o implAmbulanceWaitingListAPI) UpdateWaitingListEntry(c *gin.Context) {
	updateAmbulanceFunc(c, func(c *gin.Context, ambulance *Ambulance) (*Ambulance, interface{}, int) {
		var entry WaitingListEntry

		if err := c.ShouldBindJSON(&entry); err != nil {
			return nil, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Invalid request body",
				"error":   err.Error(),
			}, http.StatusBadRequest
		}

		entryId := c.Param("entryId")

		if entryId == "" {
			return nil, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Entry ID is required",
			}, http.StatusBadRequest
		}

		entryIndx := slices.IndexFunc(ambulance.WaitingList, func(waiting WaitingListEntry) bool {
			return entryId == waiting.Id
		})

		if entryIndx < 0 {
			return nil, gin.H{
				"status":  http.StatusNotFound,
				"message": "Entry not found",
			}, http.StatusNotFound
		}

		if entry.PatientId != "" {
			ambulance.WaitingList[entryIndx].PatientId = entry.PatientId
		}

		if entry.Id != "" {
			ambulance.WaitingList[entryIndx].Id = entry.Id
		}

		if entry.WaitingSince.After(time.Time{}) {
			ambulance.WaitingList[entryIndx].WaitingSince = entry.WaitingSince
		}

		if entry.EstimatedDurationMinutes > 0 {
			ambulance.WaitingList[entryIndx].EstimatedDurationMinutes = entry.EstimatedDurationMinutes
		}

		ambulance.reconcileWaitingList()
		return ambulance, ambulance.WaitingList[entryIndx], http.StatusOK
	})
}
