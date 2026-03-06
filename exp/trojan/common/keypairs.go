/*
Create: 2023/3/1
Project: Niphotri
Github: https://github.com/landers1037
Copyright Renj
*/

package common

// 生成双向加密的密钥 提供Base64 Des编码加解密
import (
	"Hamburger/exp/trojan/internal"
	"github.com/golang-module/dongle"
)

var cipher *dongle.Cipher

func init() {
	cipher = dongle.NewCipher()
	cipher.SetMode(dongle.CBC)      // CBC、ECB、CFB、OFB、CTR
	cipher.SetPadding(dongle.PKCS7) // No、Empty、Zero、PKCS5、PKCS7、AnsiX923、ISO97971
	cipher.SetKey(internal.KEY)     // key 长度必须是 8 字节
	cipher.SetIV(internal.KEY)      // iv 长度必须是 8 字节
}

func B64Encrypt(input string) string {
	return dongle.Encode.FromString(input).ByBase64().ToString()
}

func B64Decrypt(input string) string {
	return dongle.Decode.FromString(input).ByBase64().ToString()
}

func DesEncrypt(input string) string {
	return dongle.Encrypt.FromString(input).ByDes(cipher).ToHexString()
}

func DesDecrypt(input string) string {
	return dongle.Decrypt.FromHexString(input).ByDes(cipher).ToString()
}

func BlowfishEncrypt(input string) string {
	return dongle.Encrypt.FromString(input).ByBlowfish(cipher).ToHexString()
}

func BlowfishDecrypt(input string) string {
	return dongle.Decrypt.FromHexString(input).ByBlowfish(cipher).ToString()
}
