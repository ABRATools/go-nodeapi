package routes

import (
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sonarping/go-nodeapi/pkg/nginxtemplates"
	"github.com/sonarping/go-nodeapi/pkg/podmanapi"
)

func RegisterContainerRoutes(router *gin.Engine) {
	api := router.Group("/containers")
	{
		api.GET("/list", func(c *gin.Context) {
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			podmanContainers, err := podmanapi.ListPodmanContainers(podmanContext)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error getting Podman Containers: %v", err)
				return
			}
			if len(podmanContainers) == 0 {
				c.String(http.StatusNoContent, "No containers found")
				return
			}
			c.JSON(http.StatusOK, podmanContainers)
		})
		api.POST("/stop/:id", func(c *gin.Context) {
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			id := c.Param("id")
			status, err := podmanapi.StopPodmanContainer(podmanContext, id)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error stopping Podman Containers: %v", err)
				return
			}
			c.JSON(http.StatusOK, status)
		})
		api.POST("/start/:id", func(c *gin.Context) {
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			id := c.Param("id")
			status, err := podmanapi.StartPodmanContainer(podmanContext, id)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error starting Podman Containers: %v", err)
				return
			}
			container_ip, err := podmanapi.GetIPAddress(podmanContext, id)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error getting IP Address of Podman Containers: %v", err)
				return
			}
			container_name, err := podmanapi.GetContainerName(podmanContext, id)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error getting Container Name of Podman Containers: %v", err)
				return
			}
			// recreate nginx config
			// use default portmap for now
			webConf := nginxtemplates.NginxConfig{
				Path: container_name,
				IP:   container_ip,
				PortMap: map[uint]string{
					5801: "novnc",
					7681: "ttyd",
				},
			}
			err = nginxtemplates.GenerateNginxConfig(webConf)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error generating Nginx Config: %v", err)
				return
			}

			c.JSON(http.StatusOK, status)
		})
		type DeleteContainerRequest struct {
			EnvironmentID   string `json:"env_id" binding:"required"`
			EnvironmentName string `json:"env_name" binding:"required"`
		}
		api.POST("/remove/", func(c *gin.Context) {
			var req DeleteContainerRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			}
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			id := req.EnvironmentID
			name := req.EnvironmentName
			err = podmanapi.RemovePodmanContainer(podmanContext, id)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error removing Podman Containers: %v", err)
				return
			}
			// remove nginx config
			err = nginxtemplates.DeleteNginxConfig(name)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error removing Nginx Config: %v", err)
				return
			}

			hostname, err := os.Hostname()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error getting hostname: %v", err)
				return
			}
			baseLogDir := "/var/log/"
			logDir := filepath.Join(baseLogDir, hostname, name)
			log.Printf("Attempting to remove path: %s", logDir)
			fileStat, err := os.Stat(logDir)
			log.Printf("File stat: %v", fileStat)
			if !os.IsNotExist(err) {
				err = os.RemoveAll(logDir)
				if err != nil {
					c.String(http.StatusInternalServerError, "Error removing log directory: %v", err)
					return
				}
			}
			if err != nil {
				c.String(http.StatusInternalServerError, "Error removing log directory: %v", err)
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "Container removed successfully"})
		})

		type CreateContainerRequest struct {
			Image    string  `json:"image" binding:"required"`
			Name     string  `json:"name" binding:"required"`
			IP       string  `json:"ip"`
			CPUs     float64 `json:"cpus"`
			MemLimit int64   `json:"mem_limit"`
		}

		api.POST("/create", func(c *gin.Context) {
			var req CreateContainerRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			}
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			imageName := req.Image
			containerName := req.Name
			if imageName == "" || containerName == "" {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Image and Name are required for new environments"})
				return
			}
			exists, err := podmanapi.GetContainerName(podmanContext, containerName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			if exists != "" {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Container with name already exists"})
				return
			}
			containerID := ""
			ip := net.IP{}
			if req.IP != "" {
				// create a static IP
				ip = net.ParseIP(req.IP)
				// create the container
				containerID, err = podmanapi.CreateFromImage(podmanContext, imageName, containerName, ip, req.CPUs, req.MemLimit)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
					return
				}
			} else {
				// create the container
				containerID, err = podmanapi.CreateFromImage(podmanContext, imageName, containerName, nil, req.CPUs, req.MemLimit)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
					return
				}
			}
			// start the container
			_, err = podmanapi.StartPodmanContainer(podmanContext, containerID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			// get the container IP
			container_ip, err := podmanapi.GetIPAddress(podmanContext, containerID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			// create nginx config
			// use default portmap for now
			webConf := nginxtemplates.NginxConfig{
				Path: containerName,
				IP:   container_ip,
				PortMap: map[uint]string{
					5801: "novnc",
					7681: "ttyd",
				},
			}
			err = nginxtemplates.GenerateNginxConfig(webConf)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			c.JSON(http.StatusOK, containerID)
		})

		// expects data in form-data in the format:
		// image: <image name>
		// name: <container name>
		// ip: <static container ip> (optional)
		api.POST("/create-ebpf", func(c *gin.Context) {
			var req CreateContainerRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			}
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			imageName := req.Image
			containerName := req.Name
			if imageName == "" || containerName == "" {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Image and Name are required for new environments"})
				return
			}
			exists, err := podmanapi.GetContainerName(podmanContext, containerName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			if exists != "" {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Container with name already exists"})
				return
			}
			containerID := ""
			ip := net.IP{}
			if req.IP != "" {
				// create a static IP
				ip = net.ParseIP(req.IP)
				// create the container
				containerID, err = podmanapi.CreateEBPFContainer(podmanContext, imageName, containerName, ip, req.CPUs, req.MemLimit)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
					return
				}
			} else {
				// create the container
				containerID, err = podmanapi.CreateEBPFContainer(podmanContext, imageName, containerName, nil, req.CPUs, req.MemLimit)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
					return
				}
			}
			// start the container
			_, err = podmanapi.StartPodmanContainer(podmanContext, containerID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			// get the container IP
			container_ip, err := podmanapi.GetIPAddress(podmanContext, containerID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			// create nginx config
			// use default portmap for now
			webConf := nginxtemplates.NginxConfig{
				Path: containerName,
				IP:   container_ip,
				PortMap: map[uint]string{
					5801: "novnc",
					7681: "ttyd",
				},
			}
			err = nginxtemplates.GenerateNginxConfig(webConf)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			c.JSON(http.StatusOK, containerID)
		})
	}
}
