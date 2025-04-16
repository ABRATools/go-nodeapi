package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sonarping/go-nodeapi/pkg/podmanapi"
)

func RegisterEBPFRoutes(router *gin.Engine) {
	ebpf := router.Group("/ebpf")
	{
		ebpf.GET("/ebpf-info/:container_id", func(c *gin.Context) {
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			id := c.Param("container_id")
			EBPFServices, err := podmanapi.GetEBPFSystemdUnits(podmanContext, id)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error getting EBPF Info: %v", err)
				return
			}
			if len(EBPFServices) == 0 {
				c.String(http.StatusNoContent, "No EBPF services found")
				return
			}
			c.JSON(http.StatusOK, EBPFServices)
		})
		// expects data in form-data in the format:
		// container_id: <container id>
		// ebpf_service: <ebpf service name>

		type EBPFServiceRequest struct {
			ContainerID string `json:"container_id" binding:"required"`
			EBPFService string `json:"ebpf_service" binding:"required"`
		}

		ebpf.POST("/ebpf-start-service", func(c *gin.Context) {
			var req EBPFServiceRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			}
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			id := req.ContainerID
			ebpfService := req.EBPFService
			// Start the EBPF service
			_, err = podmanapi.StartEBPFService(podmanContext, id, ebpfService)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error starting EBPF Service: %v", err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "EBPF service started successfully"})
		})

		ebpf.POST("/ebpf-stop-service", func(c *gin.Context) {
			var req EBPFServiceRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			}
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			id := req.ContainerID
			ebpfService := req.EBPFService
			// Start the EBPF service
			_, err = podmanapi.StopEBPFService(podmanContext, id, ebpfService)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error stopping EBPF Service: %v", err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "EBPF service stopped successfully"})
		})
	}
}
