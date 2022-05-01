//程序操作
package big

import (
	"crypto/rand"
	"math"
	"math/big"
	rands "math/rand"
	"strconv"
	"time"
)

/**
取随机范围数字
传入：
	min：起始数
	max：结束数
	randtype: 单双选择，0=不限制单双，1=取单，2=取双
返回：
	随机数字
*/
func ProgRangeRand(min int, max int, randtype int) int {
	res := 0
	if min < 0 {
		f64Min := math.Abs(float64(min))
		i64Min := int64(f64Min)
		result, _ := rand.Int(rand.Reader, big.NewInt(int64(max+1)+i64Min))
		res = int(result.Int64() - i64Min)
	} else {
		result, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
		res = min + int(result.Int64())
	}
	if (randtype == 1 && res%2 == 0) || (randtype == 2 && res%2 != 0) {
		if res == max {
			res = res - 1
		} else {
			res = res + 1
		}
	}
	return res
}

/**
取随机指定位数数字
传入：
	len：欲取随机数长度
	randtype: 单双选择，0=不限制单双，1=取单，2=取双
返回：
	随机数字
*/
func ProgLenRand(lens int, randtype int) string {
	res := ""
	rands.Seed(time.Now().UnixNano())
	for i := 0; i < lens; i++ {
		ir := rands.Intn(10)
		if i == 0 && ir == 0 {
			for true {
				ir = rands.Intn(10)
				if ir != 0 {
					break
				}
			}
		}
		if i+1 == lens && randtype != 0 {
			if (randtype == 1 && ir%2 == 0) || (randtype == 2 && ir%2 != 0) {
				ir++
			}
		}
		res = res + strconv.Itoa(ir)
	}
	return res
}

/**
取随机生成指定长度字符串
传参：
	len：想要生成的长度
	typ：0=不限制、1=只生成数字、2=只生成大写字母、3=只生成小写字母
	punctuation：是否掺杂标点符号，false=不加，true=加,当typ为0时本参数有效，否则强行为false
返回：
	字符串结果
*/
func ProgRandChar(len int, typ int, punctuation bool) string {
	res := ""
	typs := typ
	for i := 0; i < len; i++ {
		if typ == 0 {
			if punctuation {
				typs = ProgRangeRand(1, 4, 0)
			} else {
				typs = ProgRangeRand(1, 3, 0)
			}
		}
		switch typs {
		case 1:
			//生成数字
			res += string(rune(ProgRangeRand(48, 57, 0)))
		case 2:
			//生成大写字母
			res += string(rune(ProgRangeRand(65, 90, 0)))
		case 3:
			//生成小写字母
			res += string(rune(ProgRangeRand(97, 122, 0)))
		case 4:
			//生成标点符号
			res += string(rune(ProgRangeRand(35, 47, 0)))
		}
	}
	return res
}

/**
四舍五入取整
传参：
	number：需要四舍五入的小数
返回：
	四舍五入后的整数，是float64类型
*/
func ProgRound(number float64) float64 {
	return math.Floor(number + 0.05)
}
