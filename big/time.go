//时间操作
package big

import (
	"strconv"
	"time"
)

/**
时间常规格式化，返回现在时间的yyyy-MM-dd hh:mm:ss 24小时制的格式字符串
传参：
	addTime：是否添加时间，为true则返回带有时分秒的字符串，否则只返回年月日
返回：
	日期时间的字符串
*/
func TimeFastGet(addTime bool) string {
	res := ""
	if addTime {
		res = time.Now().Format("2006-01-02 15:04:05")
	} else {
		res = time.Now().Format("2006-01-02")
	}
	return res
}

/**
取现行时间戳
传参：
	lens：要取的时间戳长度，秒级为10位，毫秒级为13位，纳秒级为19位.
返回2参数：
	参数1：string类型的时间戳
	参数2：int64类型的时间戳
*/
func TimeStamp(lens int) (string, int64) {
	if lens > 19 {
		lens = 19
	}
	stamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	stamp = stamp[:lens]
	stampTnt64, _ := strconv.ParseInt(stamp, 10, 64)
	return stamp, stampTnt64
}

/**
随机范围延迟
传参：
	min：起始数
	max：结束数
	duration：时间单位，毫秒=time.Millisecond，秒=time.Second，分钟=time.Minute，小时=time.Hour
*/
func TimeSleepRangeRand(min int, max int, duration time.Duration) {
	time.Sleep(time.Duration(ProgRangeRand(min, max, 0)) * duration)
}
