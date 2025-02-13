package nginxtemplates

import (
	"log"
	"os"
	"text/template"

	"github.com/sonarping/go-nodeapi/pkg/systemd"
)

const centralNginxConfigPath = "/etc/nginx/sites-enabled/abra-central.conf"
const nginxConfigPath = "/etc/nginx/snippets/abra/"

// NginxConfig is a struct that holds the configuration for the Nginx template
type NginxConfig struct {
	Path    string
	IP      string
	PortMap map[uint]string
}

func createConfigPath() error {
	err := os.MkdirAll(nginxConfigPath, os.ModePerm)
	if err != nil {
		log.Printf("Error creating directory: %v", err)
		return err
	}
	return nil
}

func generateCentralNginxConfig() error {
	// Check if the directory exists, if not create it
	_, err := os.Stat(nginxConfigPath)
	// if does not exist:
	if err != nil {
		err = createConfigPath()
		if err != nil {
			return err
		}
	}
	tmpl, err := template.New("nginx").Parse(`
server {
	listen 9999;
	server_name localhost;

	include /etc/nginx/snippets/abra/*.conf;
}
`)
	// if error when parsing template return error
	if err != nil {
		return err
	}
	templateFile, err := os.Create(centralNginxConfigPath)
	if err != nil {
		log.Printf("Error creating central nginx config: %v", err)
		return err
	}
	defer templateFile.Close()
	err = tmpl.Execute(templateFile, nil)
	if err != nil {
		log.Printf("Error executing template: %v", err)
		return err
	}
	return nil
}

func DeleteNginxConfig(path string) error {
	err := os.Remove(nginxConfigPath + path + ".conf")
	if err != nil {
		log.Printf("Error deleting nginx config: %v", err)
		return err
	}
	return nil
}

// GenerateNginxConfig generates the Nginx configuration file
func GenerateNginxConfig(nginxConfig NginxConfig) error {
	// test for central config
	_, err := os.Stat(centralNginxConfigPath)
	// if not already exists:
	if err != nil {
		err = generateCentralNginxConfig()
		if err != nil {
			return err
		}
	}
	tmpl, err := template.New("nginx").Parse(`
	{{ range $port, $endpoint := .PortMap }}
	location /{{ $.Path }}/{{ $endpoint }}/ {
		proxy_pass http://{{ $.IP }}:{{ $port }};
		proxy_set_header Host $host;
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

		proxy_http_version 1.1;
		proxy_set_header Upgrade $http_upgrade;
		proxy_set_header Connection 'upgrade';

		proxy_cache_bypass $http_upgrade;
	}
	{{ end }}
`)
	if err != nil {
		return err
	}
	_, err = os.Stat(nginxConfigPath)
	if err != nil {
		err = createConfigPath()
		if err != nil {
			return err
		}
	}
	var newConfigPath string = nginxConfigPath + nginxConfig.Path + ".conf"
	templateFile, err := os.Create(newConfigPath)
	if err != nil {
		return err
	}
	defer templateFile.Close()
	err = tmpl.Execute(templateFile, nginxConfig)
	if err != nil {
		return err
	}
	err = ReloadNginx()
	if err != nil {
		return err
	}
	return nil
}

func ReloadNginx() error {
	err := systemd.ReloadUnit("nginx.service", "replace")
	if err != nil {
		log.Fatalf("Failed to reload Nginx: %v, please make sure nginx is installed", err)
		return err
	}
	return nil
}
