package nginxtemplates

import (
	"os"
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
		t.Fatalf("expected nil, got: %v", err)
	}
	_, err = os.ReadFile("/etc/nginx/snippets/abra/fatbingus.conf")
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}

}
