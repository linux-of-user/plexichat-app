package cmd

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"net/http"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Launch the PlexiChat web interface",
	Run: func(cmd *cobra.Command, args []string) {
		r := gin.Default()
		r.Static("/static", "./web/static")
		r.LoadHTMLGlob("web/*.html")

		r.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", nil)
		})

		r.Run(":8080") // listen and serve on 0.0.0.0:8080
	},
}

func init() {
	rootCmd.AddCommand(webCmd)
}