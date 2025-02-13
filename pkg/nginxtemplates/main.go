package nginxtemplates

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/sonarping/go-nodeapi/pkg/systemd"
)

const (
	centralNginxConfigPath = "/etc/nginx/sites-enabled/abra-central.conf"
	nginxConfigDir         = "/etc/nginx/snippets/abra"
)

// NginxConfig holds the configuration for the Nginx template.
type NginxConfig struct {
	Path    string
	IP      string
	PortMap map[uint]string
}

// createConfigDir ensures that the nginx configuration directory exists.
func createConfigDir() error {
	if err := os.MkdirAll(nginxConfigDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating config directory %q: %w", nginxConfigDir, err)
	}
	return nil
}

// generateCentralNginxConfig creates the central Nginx config file if it doesn't exist.
func generateCentralNginxConfig() error {
	// Ensure the config directory exists.
	if err := createConfigDir(); err != nil {
		return err
	}

	// Define the central Nginx configuration template.
	const tmplText = `
server {
	listen 9999;
	server_name localhost;

	include /etc/nginx/snippets/abra/*.conf;
}
`
	tmpl, err := template.New("nginxCentral").Parse(tmplText)
	if err != nil {
		return fmt.Errorf("parsing central nginx template: %w", err)
	}

	// Create or truncate the central config file.
	file, err := os.Create(centralNginxConfigPath)
	if err != nil {
		return fmt.Errorf("creating central nginx config file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, nil); err != nil {
		return fmt.Errorf("executing central nginx template: %w", err)
	}

	return nil
}

// DeleteNginxConfig removes an Nginx config file based on the provided name.
func DeleteNginxConfig(name string) error {
	configPath := filepath.Join(nginxConfigDir, name+".conf")
	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("deleting nginx config %q: %w", configPath, err)
	}
	return nil
}

// GenerateNginxConfig creates an additional Nginx configuration file using the provided settings.
// If the central configuration does not exist, it will be generated.
func GenerateNginxConfig(cfg NginxConfig) error {
	// Ensure the central configuration exists.
	if _, err := os.Stat(centralNginxConfigPath); os.IsNotExist(err) {
		if err := generateCentralNginxConfig(); err != nil {
			return err
		}
	}

	// Ensure the config directory exists.
	if err := createConfigDir(); err != nil {
		return err
	}

	// Define the additional Nginx configuration template.
	const tmplText = `
{{- range $port, $endpoint := .PortMap }}
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
{{ end -}}
`
	tmpl, err := template.New("nginx").Parse(tmplText)
	if err != nil {
		return fmt.Errorf("parsing nginx config template: %w", err)
	}

	// Build the file path for the new configuration.
	newConfigPath := filepath.Join(nginxConfigDir, cfg.Path+".conf")
	file, err := os.Create(newConfigPath)
	if err != nil {
		return fmt.Errorf("creating nginx config file %q: %w", newConfigPath, err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, cfg); err != nil {
		return fmt.Errorf("executing nginx config template: %w", err)
	}

	// Reload Nginx to apply the new configuration.
	if err := ReloadNginx(); err != nil {
		return fmt.Errorf("reloading nginx: %w", err)
	}

	return nil
}

// ReloadNginx reloads the Nginx service using systemd.
func ReloadNginx() error {
	// Replace the call below with your actual systemd reload function.
	if err := systemd.ReloadUnit("nginx.service", "replace"); err != nil {
		return fmt.Errorf("failed to reload nginx: %w", err)
	}
	return nil
}
