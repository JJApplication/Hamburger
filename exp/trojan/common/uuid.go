/*
Create: 2023/3/1
Project: Niphotri
Github: https://github.com/landers1037
Copyright Renj
*/

package common

// uuid生成

import (
	uuid "github.com/satori/go.uuid"
)

func UUID() string {
	return uuid.NewV4().String()
}
