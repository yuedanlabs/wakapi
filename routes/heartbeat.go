package routes

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/muety/wakapi/services"
	"github.com/muety/wakapi/utils"

	"github.com/muety/wakapi/models"
)

type HeartbeatHandler struct {
	config        *models.Config
	heartbeatSrvc *services.HeartbeatService
}

func NewHeartbeatHandler(heartbeatService *services.HeartbeatService) *HeartbeatHandler {
	return &HeartbeatHandler{
		config:        models.GetConfig(),
		heartbeatSrvc: heartbeatService,
	}
}

type heartbeatResponseVm struct {
	Responses [][]interface{} `json:"responses"`
}

func (h *HeartbeatHandler) ApiPost(w http.ResponseWriter, r *http.Request) {
	var heartbeats []*models.Heartbeat
	user := r.Context().Value(models.UserKey).(*models.User)
	opSys, editor, _ := utils.ParseUserAgent(r.Header.Get("User-Agent"))

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&heartbeats); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	for _, hb := range heartbeats {
		hb.OperatingSystem = opSys
		hb.Editor = editor
		hb.User = user
		hb.UserID = user.ID
		hb.Augment(h.config.CustomLanguages)

		if !hb.Valid() {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid heartbeat object."))
			return
		}
	}

	if err := h.heartbeatSrvc.InsertBatch(heartbeats); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		os.Stderr.WriteString(err.Error())
		return
	}

	utils.RespondJSON(w, http.StatusCreated, constructSuccessResponse(len(heartbeats)))
}

// construct weird response format (see https://github.com/wakatime/wakatime/blob/2e636d389bf5da4e998e05d5285a96ce2c181e3d/wakatime/api.py#L288)
// to make the cli consider all heartbeats to having been successfully saved
// response looks like: { "responses": [ [ { "data": {...} }, 201 ], ... ] }
func constructSuccessResponse(n int) *heartbeatResponseVm {
	responses := make([][]interface{}, n)

	for i := 0; i < n; i++ {
		r := make([]interface{}, 2)
		r[0] = nil
		r[1] = http.StatusCreated
		responses[i] = r
	}

	return &heartbeatResponseVm{
		Responses: responses,
	}
}
