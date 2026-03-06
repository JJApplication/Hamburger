/*
Create: 2023/3/1
Project: Niphotri
Github: https://github.com/landers1037
Copyright Renj
*/

package common

import "testing"

func TestB64(t *testing.T) {
	i := "Hello"
	t.Log(B64Encrypt(i))
	t.Log(B64Decrypt(B64Encrypt(i)))
}

func TestDes(t *testing.T) {
	i := "Hello"
	t.Log(DesEncrypt(i))
	t.Log(DesDecrypt(DesEncrypt(i)))
}

func TestBlowfish(t *testing.T) {
	i := "Hello"
	t.Log(BlowfishEncrypt(i))
	t.Log(BlowfishDecrypt(BlowfishEncrypt(i)))
}
