package Commands

import (
	"fmt"
	"forklift/Lib/Config"
	"github.com/spf13/cobra"
	"strconv"
)

func init() {
	configCommand.AddCommand(configGetCommand)
	configCommand.AddCommand(configSetCommand)
	configCommand.AddCommand(configShowCommand)
	configCommand.AddCommand(configDeleteCommand)

	rootCmd.AddCommand(configCommand)
}

var configCommand = &cobra.Command{
	Use:   "config",
	Short: "Gets or sets values in config file",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var err = Config.Init(nil)
		if err != nil {
			logger.Errorf("Config error, bypassing: %s", err)
			return
		}
	},
}

var configShowCommand = &cobra.Command{
	Use:   "show",
	Short: "Show current merged config",
	Run: func(cmd *cobra.Command, args []string) {
		var value, _ = Config.AppConfig.GetAll()
		fmt.Printf("%s\n", string(value))
	},
}

var configGetCommand = &cobra.Command{
	Use:   "get <key>",
	Short: "Gets value from config file by name e.g. `forklift config get storage.type`",
	Run: func(cmd *cobra.Command, args []string) {
		var value = Config.AppConfig.Get(args[0])
		fmt.Printf("%v\n", value)
	},
}

var configDeleteCommand = &cobra.Command{
	Use:     "delete <key>",
	Aliases: []string{"unset", "remove", "rm"},
	Short:   "Unset value by name e.g. `forklift config delete general.threadsCount`",
	Run: func(cmd *cobra.Command, args []string) {
		Config.AppConfig.Delete(args[0])

		err := Config.AppConfig.Save(nil)
		if err != nil {
			fmt.Printf("Error writing config: %v\n", err)
			return
		}
	},
}

var configSetCommand = &cobra.Command{
	Use: "set <name> <value> [type]",
	Short: "Sets value from config file by name" +
		"." +
		" Available types: 'string', 'int', 'float', 'bool'" +
		" e.g. `forklift config set storage.type s3 string`",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) < 2 || len(args) > 3 {
			fmt.Printf("Args count must be 2 or 3\n")
			_ = cmd.Help()
			return
		}

		var value any = args[1]

		if len(args) == 3 {
			var err error
			var v any

			switch args[2] {
			case "float":
				v, err = strconv.ParseFloat(value.(string), 64)
				if err == nil {
					value = v
				}
			case "int":
				v, err = strconv.ParseInt(value.(string), 10, 64)
				if err == nil {
					value = v
				}
			case "bool":
				v, err = strconv.ParseBool(value.(string))
				if err == nil {
					value = v
				}
			default:
			case "string":
				value = value.(string)
			}

			if err != nil {
				fmt.Printf("Error converting value '%s' to '%s', %v\n", args[1], args[2], err)
				return
			}
		}

		var prevValue = Config.AppConfig.Get(args[0])
		fmt.Printf("previous %s: %v\n", args[0], prevValue)

		Config.AppConfig.Set(args[0], value)

		err := Config.AppConfig.Save(nil)
		if err != nil {
			fmt.Printf("Error writing config: %v\n", err)
			return
		}

		fmt.Printf("new %s: %v\n", args[0], args[1])
	},
}
