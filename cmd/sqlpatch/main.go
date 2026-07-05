package main

import (
	"bufio"
	"fmt"
	"gin-fast/app/global/app"
	"gin-fast/app/global/consts"
	"gin-fast/app/utils/gormhelper"
	"gin-fast/app/utils/ymlconfig"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

func main() {
	basePath, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	app.BasePath = basePath
	app.ZapLog = zap.NewNop()
	app.ConfigYml = ymlconfig.CreateYamlFactory(filepath.Join(basePath, "config"))

	dbType := app.ConfigYml.GetString("gormv2.usedbtype")
	switch dbType {
	case consts.DbTypeMySql:
		app.GormDbMysql, err = gormhelper.GetOneMysqlClient()
	case consts.DbTypeSqlServer:
		app.GormDbSqlserver, err = gormhelper.GetOneSqlserverClient()
	case consts.DbTypePostgreSql:
		app.GormDbPostgreSql, err = gormhelper.GetOnePostgreSqlClient()
	default:
		err = fmt.Errorf("unsupported db type: %s", dbType)
	}
	if err != nil {
		panic(err)
	}

	patchPath := filepath.Join(basePath, "resource", "database", "patch_20260705_missing_system_menus_ascii.sql")
	content, err := os.ReadFile(patchPath)
	if err != nil {
		panic(err)
	}

	statements := splitStatements(string(content))
	tx := app.DB().Begin()
	if tx.Error != nil {
		panic(tx.Error)
	}

	for _, stmt := range statements {
		if err := tx.Exec(stmt).Error; err != nil {
			tx.Rollback()
			panic(fmt.Errorf("exec failed: %w\nstatement: %s", err, stmt))
		}
	}

	if err := tx.Commit().Error; err != nil {
		panic(err)
	}

	fmt.Printf("Executed %d SQL statements successfully.\n", len(statements))
}

func splitStatements(sqlText string) []string {
	scanner := bufio.NewScanner(strings.NewReader(sqlText))
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)

	var statements []string
	var builder strings.Builder

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		upper := strings.ToUpper(line)
		if upper == "START TRANSACTION;" || upper == "COMMIT;" {
			continue
		}

		builder.WriteString(line)
		builder.WriteByte('\n')

		if strings.HasSuffix(line, ";") {
			stmt := strings.TrimSpace(builder.String())
			stmt = strings.TrimSuffix(stmt, ";")
			stmt = strings.TrimSpace(stmt)
			if stmt != "" {
				statements = append(statements, stmt)
			}
			builder.Reset()
		}
	}

	if tail := strings.TrimSpace(builder.String()); tail != "" {
		statements = append(statements, tail)
	}

	return statements
}
