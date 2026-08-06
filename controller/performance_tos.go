package controller

import (
	"net/http"

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
	if _, err := objectstorage.NewTOS(config); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "TOS configuration is valid"})
}
