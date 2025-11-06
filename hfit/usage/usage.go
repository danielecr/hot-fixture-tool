package usage

import "fmt"

func printUsagePrefix() {
	fmt.Println("hfit - Hot Fixture Tool CLI")
	fmt.Println("Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>")
	fmt.Println("GitHub: https://github.com/danielecr/hot-fixture-tool")
	fmt.Println()
	fmt.Println("Configuration: ~/.hfit/config")
	fmt.Println("JWT token: ~/.hfit/token")
	fmt.Println()
	fmt.Println("Usage:")
}
func printUsageSuffix() {
	fmt.Println()
	fmt.Println("Support: Daniele Cruciani <daniele@smartango.com>")
}

func PrintUsage() {
	printUsagePrefix()
	fmt.Println("  hfit help                                           Show this help message")
	fmt.Println("  hfit config <hfitd_host> <email> <public_key_path>  Configure connection")
	fmt.Println("  hfit login                                          Authenticate and get JWT token")
	fmt.Println("  hfit dbmss                                          List available DBMS providers")
	fmt.Println("  hfit dbs <dbms>                                     List databases for DBMS provider")
	fmt.Println("  hfit tables <dbms> <db_id>                          List tables in database")
	fmt.Println("  hfit rows <dbms> <db_id> <table_id> [filterpart]    Stream table rows as NDJSON")
	fmt.Println("  hfit files <volume> [filters...]                    Stream files as NDJSON")
	fmt.Println("  hfit pkg-tmpl list                                  List all your package templates")
	fmt.Println("  hfit pkg-tmpl show <template_name>                  Show specific template YAML")
	fmt.Println("  hfit pkg-tmpl create <template_file.yaml>           Create new template from file")
	fmt.Println("  hfit pkg-tmpl update <template_file.yaml>           Update existing template from file")
	fmt.Println("  hfit pkg-tmpl patch <template_file.yaml>            Partially update template (shows diff)")
	fmt.Println("  hfit pkg-tmpl delete <template_name>                Delete template")
	fmt.Println("  hfit pkg-download <template_name> [param1] [param2] Generate package from template")
	fmt.Println("  hfit download <file_path>                           Download file")
	printUsageSuffix()
}

func PrintUsagePkgTmpl() {
	printUsagePrefix()
	fmt.Println("Usage:")
	fmt.Println("  hfit pkg-tmpl list                           List all your package templates")
	fmt.Println("  hfit pkg-tmpl show <template_name>           Show specific template YAML")
	fmt.Println("  hfit pkg-tmpl create <template_file.yaml>    Create new template from file")
	fmt.Println("  hfit pkg-tmpl update <template_file.yaml>    Update existing template from file")
	fmt.Println("  hfit pkg-tmpl patch <template_file.yaml>     Partially update template (shows diff)")
	fmt.Println("  hfit pkg-tmpl delete <template_name>         Delete template")
	printUsageSuffix()
}
