package main

import "forklift/Commands"

func main() {

	/*
		var l = ConsoleLogger.NewLoggerWithFormatter("main", &ConsoleLogger.TextFormatter{
			Indentation: 8,
		})

		l.Log(nil, "Starting Forklift\n- Loading configuration\n- Starting services")

		l.Log(nil, `Starting Forklift
			- Loading configuration
			- Starting services`)*/

	Commands.Execute()

	return
}
