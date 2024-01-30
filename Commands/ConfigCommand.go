package Commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strconv"
)

func init() {
	configCommand.AddCommand(configGetCommand)
	configCommand.AddCommand(configSetCommand)

	rootCmd.AddCommand(configCommand)
}

var configCommand = &cobra.Command{
	Use:   "config",
	Short: "get or set values in config file",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s\n", viper.ConfigFileUsed())
	},
}

var configGetCommand = &cobra.Command{
	Use:   "get",
	Short: "Gets value from config file by name e.g. `forklift config get storage.type`",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s\n", viper.ConfigFileUsed())
		var value = viper.Get(args[0])
		fmt.Printf("%v\n", value)
	},
}

var configSetCommand = &cobra.Command{
	Use: "set <name> <value> [type]",
	Short: "Sets value from config file by name\n" +
		"Available types: 'string', 'int', 'float', 'bool'\n" +
		" e.g. `forklift config set storage.type s3 string`",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) < 2 || len(args) > 3 {
			fmt.Printf("Args count must be 2 or 3\n")
			cmd.Help()
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

		fmt.Printf("%s\n", viper.ConfigFileUsed())
		var prevValue = viper.Get(args[0])
		fmt.Printf("previous %s: %v\n", args[0], prevValue)

		viper.SetDefault(args[0], value)
		viper.Set(args[0], value)

		err := viper.WriteConfig()
		if err != nil {
			fmt.Printf("Error writing config: %v\n", err)
			return
		}

		fmt.Printf("new %s: %v\n", args[0], args[1])
	},
}
