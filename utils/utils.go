package utils

import (
	"context"
	_ "embed"
	"errors"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/CESSProject/go-keyring"
	"github.com/oschwald/geoip2-golang"
)

const baseStr = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()[]{}+-*/_=.<>?:|,~"

func GetDirSize(path string) (int64, error) {
	fs, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if fs.IsDir() {
		return fs.Size(), nil
	}
	return 0, errors.New("not dir")
}

func GetFileNum(path string) (int, error) {
	var num int
	dir, err := os.ReadDir(path)
	if err != nil {
		return num, err
	}
	for _, f := range dir {
		if !f.IsDir() {
			num++
		}
	}
	return num, nil
}

// Generate random password
func GetRandomcode(length uint8) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	bytes := make([]byte, length)
	l := len(baseStr)
	for i := uint8(0); i < length; i++ {
		bytes[i] = baseStr[r.Intn(l)]
	}
	return string(bytes)
}

func IsIPv4(ipAddr string) bool {
	ip := net.ParseIP(ipAddr)
	return ip != nil && strings.Contains(ipAddr, ".")
}

func IsRateValue(v float64) bool {
	if v >= 0 && v <= 1 {
		return true
	}
	return false
}

// Get external network ip
func GetExternalIp() (string, error) {
	var (
		err        error
		externalIp string
	)

	client := http.Client{
		Timeout: time.Duration(10 * time.Second),
	}
	resp, err := client.Get("http://myexternalip.com/raw")
	if err == nil {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		externalIp = string(b)
		if IsIPv4(externalIp) {
			return externalIp, nil
		}
	}

	ctx1, _ := context.WithTimeout(context.Background(), 10*time.Second)
	output, err := exec.CommandContext(ctx1, "bash", "-c", "curl ifconfig.co").Output()
	if err == nil {
		externalIp = strings.ReplaceAll(string(output), "\n", "")
		externalIp = strings.ReplaceAll(externalIp, " ", "")
		if IsIPv4(externalIp) {
			return externalIp, nil
		}
	}

	ctx2, _ := context.WithTimeout(context.Background(), 10*time.Second)
	output, err = exec.CommandContext(ctx2, "bash", "-c", "curl cip.cc | grep  IP | awk '{print $3;}'").Output()
	if err == nil {
		externalIp = strings.ReplaceAll(string(output), "\n", "")
		externalIp = strings.ReplaceAll(externalIp, " ", "")
		if IsIPv4(externalIp) {
			return externalIp, nil
		}
	}

	ctx3, _ := context.WithTimeout(context.Background(), 10*time.Second)
	output, err = exec.CommandContext(ctx3, "bash", "-c", `curl ipinfo.io | grep \"ip\" | awk '{print $2;}'`).Output()
	if err == nil {
		externalIp = strings.ReplaceAll(string(output), "\"", "")
		externalIp = strings.ReplaceAll(externalIp, ",", "")
		externalIp = strings.ReplaceAll(externalIp, "\n", "")
		if IsIPv4(externalIp) {
			return externalIp, nil
		}
	}
	return "", errors.New("please check your network status")
}

//go:embed GeoLite2-City.mmdb
var geoLite2 string

func ParseCountryFromIp(ip string) (string, string, error) {
	db, err := geoip2.FromBytes([]byte(geoLite2))
	if err != nil {
		return "", "", err
	}
	defer db.Close()

	record, err := db.City(net.ParseIP(ip))
	if err != nil {
		return "", "", err
	}
	return record.Country.Names["en"], record.City.Names["en"], nil
}

func VerifySign(acc string, data []byte, sign []byte) bool {
	if len(sign) < 64 {
		return false
	}
	verkr, _ := keyring.FromURI(acc, keyring.NetSubstrate{})
	var arr [64]byte
	for i := 0; i < 64; i++ {
		arr[i] = sign[i]
	}
	return verkr.Verify(verkr.SigningContext(data), arr)
}
