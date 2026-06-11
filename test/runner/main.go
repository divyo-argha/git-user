package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type PackageStat struct {
	Name     string
	Status   string // PASS, FAIL, SKIP
	Duration string
	Coverage string
}

func main() {
	fmt.Println("\033[1;36m┌─────────────────────────────────────────────────────────────────────────────┐\033[0m")
	fmt.Println("\033[1;36m│                          GIT-USER TEST SUITE RUNNER                         │\033[0m")
	fmt.Println("\033[1;36m└─────────────────────────────────────────────────────────────────────────────┘\033[0m")
	fmt.Println()

	// Clean up old coverage profiles
	_ = os.Remove("coverage.out")

	startTime := time.Now()

	// Run go test with coverage profile
	cmd := exec.Command("go", "test", "-coverprofile=coverage.out", "./...")
	
	// We want to capture stderr and stdout
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("\033[1;31m✖ Failed to create pipe: %v\033[0m\n", err)
		os.Exit(1)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Printf("\033[1;31m✖ Failed to start tests: %v\033[0m\n", err)
		os.Exit(1)
	}

	var stats []PackageStat
	var failedTests []string
	
	scanner := bufio.NewScanner(stdoutPipe)
	
	// Regex patterns
	// ok  	github.com/divyo-argha/git-user/cmd	19.167s	coverage: 36.0% of statements
	okRegex := regexp.MustCompile(`^ok\s+([^\s]+)\s+([\d\.]+s|\(cached\))(?:[\s\(\)a-zA-Z\:]+([\d\.]+%))?`)
	// FAIL	github.com/divyo-argha/git-user/cmd	4.477s
	failRegex := regexp.MustCompile(`^FAIL\s+([^\s]+)\s+([\d\.]+s)`)
	// ?   	github.com/divyo-argha/git-user/logo	[no test files]
	skipRegex := regexp.MustCompile(`^\?\s+([^\s]+)\s+\[no test files\]`)
	
	// Test failure tracker (looks for lines starting with "--- FAIL:")
	failTestRegex := regexp.MustCompile(`^--- FAIL:\s+([^\s]+)`)

	for scanner.Scan() {
		line := scanner.Text()
		
		// If it's a test run output line, print it dimmed to show progress but not clutter
		if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "=== ") {
			if strings.Contains(line, "FAIL") {
				fmt.Printf("\033[31m%s\033[0m\n", line)
			}
		}

		if matches := failTestRegex.FindStringSubmatch(line); len(matches) > 1 {
			failedTests = append(failedTests, matches[1])
		}

		// Parse package summary lines
		if matches := okRegex.FindStringSubmatch(line); len(matches) > 1 {
			name := simplifyPackageName(matches[1])
			dur := matches[2]
			cov := "0.0%"
			if len(matches) > 3 && matches[3] != "" {
				cov = matches[3]
			} else {
				cov = "-"
			}
			stats = append(stats, PackageStat{
				Name:     name,
				Status:   "PASS",
				Duration: dur,
				Coverage: cov,
			})
		} else if matches := failRegex.FindStringSubmatch(line); len(matches) > 1 {
			name := simplifyPackageName(matches[1])
			dur := matches[2]
			stats = append(stats, PackageStat{
				Name:     name,
				Status:   "FAIL",
				Duration: dur,
				Coverage: "0.0%",
			})
		} else if matches := skipRegex.FindStringSubmatch(line); len(matches) > 1 {
			name := simplifyPackageName(matches[1])
			stats = append(stats, PackageStat{
				Name:     name,
				Status:   "SKIP",
				Duration: "0.00s",
				Coverage: "-",
			})
		}
	}

	testErr := cmd.Wait()
	elapsedTotal := time.Since(startTime)

	// Fetch total statement coverage
	totalCoverage := "0.0%"
	if _, err := os.Stat("coverage.out"); err == nil {
		covCmd := exec.Command("go", "tool", "cover", "-func=coverage.out")
		covOut, err := covCmd.Output()
		if err == nil {
			covLines := strings.Split(string(covOut), "\n")
			for _, line := range covLines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "total:") {
					parts := strings.Fields(line)
					if len(parts) >= 3 {
						totalCoverage = parts[2]
					}
				}
			}
		}
	}

	// Render table
	fmt.Println()
	fmt.Println("  \033[1;37mPACKAGE                                          STATUS      TIME       COVERAGE\033[0m")
	fmt.Println("  \033[38;5;244m─────────────────────────────────────────────────────────────────────────────\033[0m")
	
	for _, stat := range stats {
		var statusStr, covStr, durStr string
		
		// Align package name
		pkgName := fmt.Sprintf("%-48s", stat.Name)
		
		switch stat.Status {
		case "PASS":
			statusStr = "\033[1;32m✔ PASS\033[0m    "
			covStr = fmt.Sprintf("\033[32m%s\033[0m", stat.Coverage)
		case "FAIL":
			statusStr = "\033[1;31m✘ FAIL\033[0m    "
			covStr = "\033[31m0.0%\033[0m"
		case "SKIP":
			statusStr = "\033[38;5;244m- SKIP\033[0m    "
			covStr = "\033[38;5;244m[no tests]\033[0m"
		}
		
		durStr = fmt.Sprintf("%-10s", stat.Duration)
		
		fmt.Printf("  %s %s %s %s\n", pkgName, statusStr, durStr, covStr)
	}
	
	fmt.Println("  \033[38;5;244m─────────────────────────────────────────────────────────────────────────────\033[0m")
	
	if len(failedTests) > 0 {
		fmt.Println()
		fmt.Println("  \033[1;31mFailed Tests:\033[0m")
		for _, ft := range failedTests {
			fmt.Printf("    \033[31m• %s\033[0m\n", ft)
		}
	}
	
	fmt.Println()
	fmt.Printf("  \033[1;37mTotal Execution Time:\033[0m  %.2fs\n", elapsedTotal.Seconds())
	fmt.Printf("  \033[1;37mTotal Statement Coverage:\033[0m \033[1;36m%s\033[0m\n", totalCoverage)
	fmt.Println()
	
	if testErr == nil && len(failedTests) == 0 {
		fmt.Println("  \033[1;32m✔ ALL TESTS PASSED SUCCESSFULLY!\033[0m")
		fmt.Println()
		os.Exit(0)
	} else {
		fmt.Println("  \033[1;31m✘ SOME TESTS FAILED.\033[0m")
		fmt.Println()
		os.Exit(1)
	}
}

func simplifyPackageName(full string) string {
	prefix := "github.com/divyo-argha/git-user"
	if full == prefix {
		return "."
	}
	return strings.TrimPrefix(full, prefix+"/")
}
