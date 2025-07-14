package pkg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

func CreateBinLinks(pkgDir string) error {
	packageJSONPath := filepath.Join(pkgDir, "package.json")
	content, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return err
	}

	var pkgMeta map[string]interface{}
	if err := json.Unmarshal(content, &pkgMeta); err != nil {
		return err
	}

	binField, ok := pkgMeta["bin"]
	if !ok {
		return nil
	}

	binMap := make(map[string]string)
	switch binVal := binField.(type) {
	case string:
		if name, ok := pkgMeta["name"].(string); ok {
			binMap[name] = binVal
		}
	case map[string]interface{}:
		for k, v := range binVal {
			if s, ok := v.(string); ok {
				binMap[k] = s
			}
		}
	default:
		return nil
	}

	binDir := filepath.Join("node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	for binName, binRelPath := range binMap {
		fullBinPath := filepath.Join(pkgDir, binRelPath)
		binLink := filepath.Join(binDir, binName)

		// Compute relative path from .bin to target script
		relTarget, err := filepath.Rel(binDir, fullBinPath)
		if err != nil {
			return err
		}
		relTarget = filepath.ToSlash(relTarget) // Ensure POSIX-style slashes

		// Remove any existing scripts
		os.Remove(binLink)
		os.Remove(binLink + ".cmd")
		os.Remove(binLink + ".ps1")

		if runtime.GOOS == "windows" {
			// Special case for nodemon
			if binName == "nodemon" {
				cmdContent := `@ECHO off
		GOTO start
		:find_dp0
		SET dp0=%~dp0
		EXIT /b
		:start
		SETLOCAL
		CALL :find_dp0
		
		IF EXIST "%dp0%\node.exe" (
		  SET "_prog=%dp0%\node.exe"
		) ELSE (
		  SET "_prog=node"
		  SET PATHEXT=%PATHEXT:;.JS;=;%
		)
		
		endLocal & goto #_undefined_# 2>NUL || title %COMSPEC% & "%_prog%"  "%dp0%\..\nodemon\bin\nodemon.js" %*
		`
				if err := os.WriteFile(binLink+".cmd", []byte(cmdContent), 0755); err != nil {
					return err
				}
			} else {
				// General CMD wrapper
				cmdContent := `@IF EXIST "%~dp0\node.exe" (
		  "%~dp0\node.exe" "%~dp0\` + relTarget + `" %*
		) ELSE (
		  node "%~dp0\` + relTarget + `" %*
		)
		exit /b %ERRORLEVEL%
		`
				if err := os.WriteFile(binLink+".cmd", []byte(cmdContent), 0755); err != nil {
					return err
				}
			}

			// PowerShell wrapper (same for all binaries)
			ps1Content := `if (Test-Path "$PSScriptRoot\node.exe") {
		  & "$PSScriptRoot\node.exe" "$PSScriptRoot/` + relTarget + `" $args
		} else {
		  & node "$PSScriptRoot/` + relTarget + `" $args
		}
		exit $LASTEXITCODE
		`
			if err := os.WriteFile(binLink+".ps1", []byte(ps1Content), 0755); err != nil {
				return err
			}
		}

	}

	return nil
}
