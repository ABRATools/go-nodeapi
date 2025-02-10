package nginxtemplates

import (
	"testing"
)

func TestTemplateCreation(t *testing.T) {
	portmap := map[uint]string{
		5801: "novnc",
		7681: "ttyd",
	}
	nginxConfig := NginxConfig{
		Path:    "test",
		IP:      "192.168.200.2",
		PortMap: portmap,
	}
	err := GenerateNginxConfig(nginxConfig)
	if err != nil {
		panic(err)
	}
	err = ReloadNginx()
	if err != nil {
		panic(err)
	}
}
