package handlerAction

import (
	"blog_api/src/model"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type SystemHandler struct{}

// Restart handles POST /api/action/system/restart requests.
// It returns first, then exits the current process so an external supervisor can restart it.
func (h *SystemHandler) Restart(c *gin.Context) {
	c.JSON(http.StatusAccepted, model.ApiResponse{
		Code:    http.StatusAccepted,
		Message: "restart scheduled",
		Data: gin.H{
			"detail": "process will exit shortly; please rely on an external supervisor to start it again",
		},
	})

	go func() {
		time.Sleep(500 * time.Millisecond)
		log.Panicf("[restart]收到人为重启请求")
		os.Exit(0)
	}()
}
