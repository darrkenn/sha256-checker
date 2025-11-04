package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	gatewayHome := "/home/gateway"
	gatewayAllowedFiles := [2]string{"pf.conf", "relayd.conf"}
	gatewayDirs := map[string]string{
		"pf.conf":     "pf",
		"relayd.conf": "relayd",
	}

	proxmoxDellHome := "/home/proxmox-dell"
	promoxDellAllowedFiles := [1]string{"storage.cfg"}
	proxmoxDellDirs := map[string]string{
		"storage.cfg": "storage-cfg",
	}

	reverseProxyHome := "/home/reverse-proxy"
	reverseProxyAllowedFiles := [3]string{"pf.conf", "relayd.conf", "httpd.conf"}
	reverseProxyDirs := map[string]string{
		"pf.conf":     "pf",
		"relayd.conf": "relayd",
		"httpd.conf":  "httpd",
	}

	r := gin.Default()

	r.GET("/proxmox-dell", computeHash(proxmoxDellHome, proxmoxDellDirs, promoxDellAllowedFiles[:]))

	r.GET("/proxmox-dell/lxc", computeHashLXC(proxmoxDellHome))

	r.GET("/gateway", computeHash(gatewayHome, gatewayDirs, gatewayAllowedFiles[:]))

	r.GET("/reverse-proxy", computeHash(reverseProxyHome, reverseProxyDirs, reverseProxyAllowedFiles[:]))

	err := r.Run(":3119")
	if err != nil {
		log.Fatal(err)
		return
	}
}

func getLatestFile(dir string) (string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var latestFile string
	var latestTime time.Time

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestFile = filepath.Join(dir, file.Name())
		}
	}

	if latestFile == "" {
		return "", fmt.Errorf("cant find file")
	}
	return latestFile, nil
}

func isAllowedFile(fileName string, allowedFiles []string) bool {
	if slices.Contains(allowedFiles, fileName) {
		return true
	} else {
		return false
	}
}

func getDir(fileName string, dirs map[string]string) (string, error) {
	if dir, exists := dirs[fileName]; exists {
		return dir, nil
	} else {
		return "", fmt.Errorf("cant find dir")
	}
}

func computeHash(homeDir string, allowedDirs map[string]string, allowedFiles []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("file")
		if query == "" {
			c.JSON(400, gin.H{"error": "file empty"})
			return
		}

		if !isAllowedFile(query, allowedFiles[:]) {
			c.JSON(400, gin.H{"error": "unsupported file"})
			return
		}
		dir, err := getDir(query, allowedDirs)
		if err != nil {
			logError(err.Error())
			c.JSON(500, gin.H{
				"error": "check logs",
			})
			return
		}

		location := fmt.Sprintf("%s/%s", homeDir, dir)

		latestFile, err := getLatestFile(location)
		if err != nil {
			c.JSON(500, gin.H{
				"error": "check logs",
			})
			return
		}

		file, err := os.Open(latestFile)
		if err != nil {
			logError(err.Error())
			c.JSON(500, gin.H{
				"error": "check logs",
			})
			return
		}
		defer file.Close()

		hash := sha256.New()

		if _, err := io.Copy(hash, file); err != nil {
			logError(err.Error())
			c.JSON(500, gin.H{
				"error": "check logs",
			})
			return
		}

		checksum := hash.Sum(nil)

		c.JSON(200, gin.H{
			"sha256sum": fmt.Sprintf("%x", checksum),
		})
	}
}

func computeHashLXC(homeDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("id")
		if query == "" {
			c.JSON(400, gin.H{"error": "id empty"})
			return
		}

		dir := fmt.Sprintf("%s/lxc/%s", homeDir, query)

		latestFile, err := getLatestFile(dir)
		if err != nil {
			logError(err.Error())
			c.JSON(500, gin.H{
				"error": "check logs",
			})
			return
		}

		file, err := os.Open(latestFile)
		if err != nil {
			logError(err.Error())
			c.JSON(500, gin.H{
				"error": "check logs",
			})
			return
		}
		defer file.Close()

		hash := sha256.New()

		if _, err := io.Copy(hash, file); err != nil {
			logError(err.Error())
			c.JSON(500, gin.H{
				"error": "check logs",
			})
			return
		}

		checksum := hash.Sum(nil)

		c.JSON(200, gin.H{
			"sha256sum": fmt.Sprintf("%x", checksum),
		})
	}
}

func logError(errorString string) {
	file, err := os.OpenFile("/var/log/sha256-check.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		return
	}
	defer file.Close()

	time := time.Now().Format("15:04:05")
	entry := fmt.Sprintf("%s: %s\n", time, errorString)

	if _, err := file.WriteString(entry); err != nil {
		return
	}
}
