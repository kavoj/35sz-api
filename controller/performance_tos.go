package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common/objectstorage"
	"github.com/QuantumNous/new-api/setting/performance_setting"
	"github.com/gin-gonic/gin"
)

func TestPerformanceTOS(c *gin.Context) {
	config := performance_setting.GetTOSConfig()
	if !config.Enabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "TOS is disabled"})
		return
	}
	storage, err := objectstorage.NewTOS(config)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	if err := storage.Check(ctx); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "TOS bucket check failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "TOS connection is valid"})
}
