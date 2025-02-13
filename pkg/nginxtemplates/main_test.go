package nginxtemplates

import (
	"testing"
)

func TestTemplateCreation(t *testing.T) {
	webConf := NginxConfig{
		Path: "fatbingus",
		IP:   "10.88.0.4",
		PortMap: map[uint]string{
			5801: "novnc",
			7681: "ttyd",
		},
	}
	err := GenerateNginxConfig(webConf)
	if err != nil {
		t.Errorf("Error creating Nginx config: %v", err)
		return
	}
}
