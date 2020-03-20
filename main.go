package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"strings"
)

var (
	username string
	password string
	address  string
	db       string
)

func init() {
	flag.StringVar(&username, "u", "root", "数据库用户名")
	flag.StringVar(&password, "p", "123456", "数据库密码")
	flag.StringVar(&address, "a", "localhost:3306", "数据库地址")
	flag.StringVar(&db, "d", "test", "数据库名称")
	flag.Parse()
}

type Filed struct {
	TableName     string
	ColumnName    string
	IsNullAble    string
	ColumnType    string
	ColumnDefault *string
	ColumnComment string
	ColumnKey     string
}

/*
	1、连接数据库
	2、读取数据库注释
	3、生成md文档
*/
func main() {
	sqlString := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", username, password, address, db)
	conn, err := sql.Open("mysql", sqlString)
	if err != nil {
		log.Println("数据库连接失败,错误:", err.Error())
		os.Exit(1)
	}
	rows, err := conn.Query("show tables")
	if err != nil {
		log.Println("sql:show tables err:", err.Error())
		os.Exit(1)
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			log.Println("scan table failed,err:", err.Error())
			os.Exit(1)
		}
		tables = append(tables, table)
	}
	infoRows, err := conn.Query(fmt.Sprintf("select TABLE_NAME,COLUMN_NAME,IS_NULLABLE, COLUMN_TYPE,COLUMN_DEFAULT,COLUMN_COMMENT,COLUMN_KEY  from information_schema.`COLUMNS` where TABLE_SCHEMA ='%s' and TABLE_NAME IN (%s)", db, "'"+strings.Join(tables, "','")+"'"))
	if err != nil {
		log.Println("select table info failed,err :", err.Error())
		os.Exit(1)
	}
	defer infoRows.Close()
	var columns []Filed
	for infoRows.Next() {
		var column Filed
		if err := infoRows.Scan(&column.TableName, &column.ColumnName, &column.IsNullAble, &column.ColumnType, &column.ColumnDefault, &column.ColumnComment, &column.ColumnKey); err != nil {
			log.Println("scan table failed,err:", err.Error())
			os.Exit(1)
		}
		columns = append(columns, column)
	}
	columnMap := map[string][]Filed{}
	for _, l := range columns {
		columnMap[l.TableName] = append(columnMap[l.TableName], l)
	}
	//查询表注释
	rowDesc, err := conn.Query("SELECT TABLE_NAME ,table_comment FROM information_schema.`TABLES` WHERE table_schema = ? ORDER BY table_name", db)
	defer rowDesc.Close()
	if err != nil {
		log.Println("query table desc failed,err:", err.Error())
		os.Exit(1)
	}
	descMap := map[string]string{}
	for rowDesc.Next() {
		var tableName, tableDesc string
		if err := rowDesc.Scan(&tableName, &tableDesc); err != nil {
			log.Println("scan table desc failed,err:", err.Error())
			os.Exit(1)
		}
		descMap[tableName] = tableDesc
	}
	createMd(tables, columnMap, descMap)
}

func createMd(list []string, listMap map[string][]Filed, descMap map[string]string) {
	filename := db + "数据字典.md"
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		log.Println("创建文件失败,错误:", err.Error())
		os.Exit(1)
	}
	defer file.Close()
	buf := bytes.Buffer{}
	//写标题
	buf.WriteString(fmt.Sprintf("# %s 数据字典\n", db))
	buf.WriteString("\n")
	for i, l := range list {
		//表名
		buf.WriteString(fmt.Sprintf("### %d. Table：%s\n", i+1, l))
		buf.WriteString("- - -\n")
		//表注释
		buf.WriteString(fmt.Sprintf("- 表注释: %s\n", descMap[l]))
		//表内容
		buf.WriteString("| 字段 | 类型 | 键 | 空 | 默认 | 注释 |\n")
		buf.WriteString("| :--: | :--: | :--: | :--: | :--: | :--: |\n")
		fields := listMap[l]
		for _, j := range fields {
			var Def string
			if j.ColumnDefault == nil {
				Def = "Null"
			} else {
				Def = *j.ColumnDefault
			}
			buf.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n", j.ColumnName, j.ColumnType, j.ColumnKey, j.IsNullAble, Def, j.ColumnComment))
		}
		buf.WriteString("\n")
	}
	if _, err := file.Write(buf.Bytes()); err != nil {
		log.Println("file write fail,err:", err.Error())
		os.Exit(1)
	}
	fmt.Println("导出成功 ==>", filename)
}
