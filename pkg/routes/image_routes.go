package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sonarping/go-nodeapi/pkg/podmanapi"
)

func RegisterImageRoutes(router *gin.Engine) {
	api := router.Group("/images")
	{
		api.GET("/list", func(c *gin.Context) {
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			podmanImages, err := podmanapi.GetImageList(podmanContext)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error getting Podman Images: %v", err)
				return
			}
			if len(podmanImages) == 0 {
				c.String(http.StatusNoContent, "No images found")
				return
			}
			c.JSON(http.StatusOK, podmanImages)
		})
		api.POST("/remove/:id", func(c *gin.Context) {
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			id := c.Param("id")
			status, err := podmanapi.RemoveImage(podmanContext, id)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error removing Podman Images: %v", err)
				return
			}
			c.JSON(http.StatusOK, status)
		})
		// api.POST("/build", func(c *gin.Context) {
		// 	// expects data in form-data in the format:
		// 	// dockerfile: <dockerfile content>
		// 	// label: <image label>
		// 	podmanContext, err := podmanapi.InitPodmanConnection()
		// 	if err != nil {
		// 		c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
		// 	}
		// 	dockerFile := c.PostForm("dockerfile")
		// 	imageLabel := c.PostForm("label")
		// 	if dockerFile == "" || imageLabel == "" {
		// 		c.String(http.StatusBadRequest, "Dockerfile and Label are required")
		// 		return
		// 	}
		// 	imageID, err := podmanapi.BuildFromDockerFile(podmanContext, dockerFile, imageLabel)
		// 	if err != nil {
		// 		c.String(http.StatusInternalServerError, "Error building Podman Images: %v", err)
		// 		return
		// 	}
		// 	c.JSON(http.StatusOK, imageID)
		// })
	}
}
