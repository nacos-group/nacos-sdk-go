package security

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

const (
	PREFIX                 = "aliyun_v4"
	CONSTANT               = "aliyun_v4_request"
	V4_SIGN_DATE_FORMATTER = "20060102"
	SIGNATURE_V4_PRODUCE   = "mse"
)

func signWithHmacSha1Encrypt(encryptText, encryptKey string) string {
	key := []byte(encryptKey)
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(encryptText))
	expectedMAC := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(expectedMAC)
}

func Sign(data, key string) (string, error) {
	signature, err := sign([]byte(data), []byte(key))
	if err != nil {
		return "", fmt.Errorf("unable to calculate a request signature: %w", err)
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

// sign 方法用于生成签名字节数组
func sign(data, key []byte) ([]byte, error) {
	mac := hmac.New(sha1.New, key)
	if _, err := mac.Write(data); err != nil {
		return nil, err
	}
	return mac.Sum(nil), nil
}

func finalSigningKeyStringWithDefaultInfo(secret, region string) (string, error) {
	signDate := time.Now().UTC().Format(V4_SIGN_DATE_FORMATTER)
	return finalSigningKeyString(secret, signDate, region, SIGNATURE_V4_PRODUCE)
}

func finalSigningKeyString(secret, date, region, productCode string) (string, error) {
	finalKey, err := finalSigningKey(secret, date, region, productCode)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(finalKey), nil
}

func finalSigningKey(secret, date, region, productCode string) ([]byte, error) {
	secondSignkey, err := regionSigningKey(secret, date, region)
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, secondSignkey)
	_, err = mac.Write([]byte(productCode))
	if err != nil {
		return nil, err
	}
	thirdSigningKey := mac.Sum(nil)

	mac = hmac.New(sha256.New, thirdSigningKey)
	_, err = mac.Write([]byte(CONSTANT))
	if err != nil {
		return nil, err
	}
	return mac.Sum(nil), nil
}

func regionSigningKey(secret, date, region string) ([]byte, error) {
	firstSignkey, err := firstSigningKey(secret, date)
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, firstSignkey)
	_, err = mac.Write([]byte(region))
	if err != nil {
		return nil, err
	}
	return mac.Sum(nil), nil
}

func firstSigningKey(secret, date string) ([]byte, error) {
	mac := hmac.New(sha256.New, []byte(PREFIX+secret))
	_, err := mac.Write([]byte(date))
	if err != nil {
		return nil, err
	}
	return mac.Sum(nil), nil
}
