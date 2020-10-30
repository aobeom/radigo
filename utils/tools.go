package utils

import (
	"github.com/aobeom/minitools"
)

// AESSuite 初始化
var AESSuite *minitools.AESSuiteBasic

// DataSuite 初始化
var DataSuite *minitools.DataSuiteBasic

// FileSuite 初始化
var FileSuite *minitools.FileSuiteBasic

// TimeSuite 初始化
var TimeSuite *minitools.TimeSuiteBasic

func init() {
	AESSuite = minitools.AESSuite()
	DataSuite = minitools.DataSuite()
	FileSuite = minitools.FileSuite()
	TimeSuite = minitools.TimeSuite()
}
