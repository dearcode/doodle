package ssh

import (
	"bytes"
	"io/ioutil"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestPublicKeys(t *testing.T) {
	//	var hostKey ssh.PublicKey

	key, err := ioutil.ReadFile("/home/tian/.ssh/id_rsa_38")
	if err != nil {
		t.Fatalf("unable to read private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		t.Fatalf("unable to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		//		HostKeyCallback: ssh.FixedHostKey(hostKey),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", "192.168.177.208:22", config)
	if err != nil {
		t.Fatalf("unable to connect: %v", err)
	}
	defer client.Close()

	s, err := client.NewSession()
	if err != nil {
		t.Fatalf("NewSession error:%v", err)
	}
	var bufErr, bufOut bytes.Buffer
	s.Stdout = &bufOut
	s.Stderr = &bufErr
	if err = s.Run("hostname"); err != nil {
		t.Fatalf("Run error:%v", err)
	}

	t.Logf("out:%v, err:%v", bufOut.String(), bufErr.String())
}
