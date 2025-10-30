/*
 * Hot Fixture Tool Daemon - CLI Administration Tool
 * Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
 *
 * This file is part of the Hot Fixture Tool project.
 * GitHub: https://github.com/danielecr/hot-fixture-tool
 *
 * Licensed under the terms specified in the LICENSE file.
 */

// hfitd-cli is a command line tool for administering hfitd server
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type AdminCommand struct {
	Action string   `json:"action"`
	Args   []string `json:"args"`
}

type AdminResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

const defaultSocketPath = "/tmp/hfitd.sock"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	socketPath := getSocketPath()
	command := os.Args[1]

	switch command {
	case "help", "-h", "--help":
		printUsage()

	case "adduser":
		if len(os.Args) != 4 {
			fmt.Println("Usage: hfitd-cli adduser <email> <public_key_file_or_content>")
			os.Exit(1)
		}
		handleAddUser(socketPath, os.Args[2], os.Args[3])

	case "renew-jwt":
		handleRenewJWT(socketPath)

	case "get-jwt-public-key":
		handleGetJWTPublicKey(socketPath)

	default:
		fmt.Printf("Error: Unknown command '%s'\n\n", command)
		fmt.Printf("Run 'hfitd-cli help' for usage information.\n")
		fmt.Printf("For support, contact: Daniele Cruciani <daniele@smartango.com>\n")
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("hfitd-cli - Administration tool for Hot Fixture Tool Daemon")
	fmt.Println("Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>")
	fmt.Println("GitHub: https://github.com/danielecr/hot-fixture-tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  hfitd-cli help                                      Show this help message")
	fmt.Println("  hfitd-cli adduser <email> <public_key_file_or_content>")
	fmt.Println("  hfitd-cli renew-jwt")
	fmt.Println("  hfitd-cli get-jwt-public-key")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  HFITD_SOCKET_PATH - Path to Unix socket (default: /tmp/hfitd.sock)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  hfitd-cli adduser alice@example.com alice_public_key.pem")
	fmt.Println("  hfitd-cli adduser bob@company.com \"-----BEGIN PUBLIC KEY-----...\"")
	fmt.Println()
	fmt.Println("For support, contact: Daniele Cruciani <daniele@smartango.com>")
	fmt.Println("Project repository: https://github.com/danielecr/hot-fixture-tool")
	fmt.Println("  hfitd-cli renew-jwt")
}

func getSocketPath() string {
	if path := os.Getenv("HFITD_SOCKET_PATH"); path != "" {
		return path
	}
	return defaultSocketPath
}

func handleAddUser(socketPath, email, publicKeyArg string) {
	publicKey := getPublicKey(publicKeyArg)

	cmd := AdminCommand{
		Action: "adduser",
		Args:   []string{email, publicKey},
	}

	response := sendCommand(socketPath, cmd)
	if response.Success {
		fmt.Printf("✓ %s\n", response.Message)
	} else {
		fmt.Printf("✗ %s\n", response.Message)
		os.Exit(1)
	}
}

func handleRenewJWT(socketPath string) {
	cmd := AdminCommand{
		Action: "renew-jwt",
		Args:   []string{},
	}

	response := sendCommand(socketPath, cmd)
	if response.Success {
		fmt.Printf("✓ %s\n", response.Message)
	} else {
		fmt.Printf("✗ %s\n", response.Message)
		os.Exit(1)
	}
}

func handleGetJWTPublicKey(socketPath string) {
	cmd := AdminCommand{
		Action: "get-jwt-public-key",
		Args:   []string{},
	}

	response := sendCommand(socketPath, cmd)
	if response.Success {
		fmt.Printf("JWT Public Key:\n%s\n", response.Data)
	} else {
		fmt.Printf("✗ %s\n", response.Message)
		os.Exit(1)
	}
}

func getPublicKey(arg string) string {
	// Check if it's a file path
	if !strings.Contains(arg, "BEGIN PUBLIC KEY") {
		// Treat as file path
		if !filepath.IsAbs(arg) {
			// Make relative to current directory
			wd, _ := os.Getwd()
			arg = filepath.Join(wd, arg)
		}

		content, err := ioutil.ReadFile(arg)
		if err != nil {
			fmt.Printf("✗ Failed to read public key file: %v\n", err)
			os.Exit(1)
		}
		return string(content)
	}

	// Treat as direct public key content
	return arg
}

func sendCommand(socketPath string, cmd AdminCommand) AdminResponse {
	// Connect to Unix socket
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return AdminResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to connect to hfitd socket (%s): %v", socketPath, err),
		}
	}
	defer conn.Close()

	// Send command
	cmdJSON, _ := json.Marshal(cmd)
	conn.Write(append(cmdJSON, '\n'))

	// Read response
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return AdminResponse{
			Success: false,
			Message: "Failed to read response from server",
		}
	}

	var response AdminResponse
	if err := json.Unmarshal(scanner.Bytes(), &response); err != nil {
		return AdminResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to parse server response: %v", err),
		}
	}

	return response
}
