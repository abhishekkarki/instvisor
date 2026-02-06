package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/abhishekkarki/instvisor/pkg/analyzer"
	"github.com/abhishekkarki/instvisor/pkg/storage"
)

var (
	dbPath     = flag.String("db", "/var/lib/instvisor/metrics.db", "Path to metrics database")
	days       = flag.Int("days", 7, "Number of days to analyze")
	currentCPU = flag.Int("current-cpu", 0, "Current number of vCPUs (optional)")
	currentMem = flag.Float64("current-mem", 0, "Current memory in GB (optional)")
	format     = flag.String("format", "text", "Output format: text, json")
)

func main() {
	flag.Parse()

	// Open storage
	store, err := storage.NewSQLiteStorage(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer store.Close()

	// Create analyzer
	an := analyzer.NewAnalyzer(store)

	// Analyze period
	period := time.Duration(*days) * 24 * time.Hour
	fmt.Printf("Analyzing metrics from the past %d days...\n\n", *days)

	analysis, err := an.AnalyzePeriod(period)
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	// Print analysis results
	printAnalysis(analysis)

	// Generate recommendation
	rec := analyzer.NewRecommender(an)
	config := &analyzer.RecommendationConfig{
		CPUHeadroomPercent:    20,
		MemoryHeadroomPercent: 15,
		UseP95:                true,
		CurrentCPUCores:       *currentCPU,
		CurrentMemoryGB:       *currentMem,
	}

	recommendation, err := rec.GenerateRecommendation(analysis, config)
	if err != nil {
		log.Fatalf("Failed to generate recommendation: %v", err)
	}

	// Print recommendation
	printRecommendation(recommendation)
}

func printAnalysis(analysis *analyzer.ResourceAnalysis) {
	fmt.Println("=== RESOURCE ANALYSIS ===")
	fmt.Printf("Analysis Period: %s\n", analysis.Period)
	fmt.Printf("Workload Pattern: %s (confidence: %.0f%%)\n\n",
		analysis.Pattern, analysis.Confidence*100)

	if analysis.CPU != nil {
		fmt.Println("CPU Usage:")
		fmt.Printf("  Mean:   %.1f%%\n", analysis.CPU.Mean)
		fmt.Printf("  P50:    %.1f%%\n", analysis.CPU.P50)
		fmt.Printf("  P90:    %.1f%%\n", analysis.CPU.P90)
		fmt.Printf("  P95:    %.1f%%\n", analysis.CPU.P95)
		fmt.Printf("  P99:    %.1f%%\n", analysis.CPU.P99)
		fmt.Printf("  Max:    %.1f%%\n", analysis.CPU.Max)
		fmt.Printf("  StdDev: %.1f\n", analysis.CPU.StdDev)
		fmt.Println()
	}

	if analysis.Memory != nil {
		fmt.Println("Memory Usage:")
		fmt.Printf("  Mean:   %.1f%%\n", analysis.Memory.Mean)
		fmt.Printf("  P50:    %.1f%%\n", analysis.Memory.P50)
		fmt.Printf("  P90:    %.1f%%\n", analysis.Memory.P90)
		fmt.Printf("  P95:    %.1f%%\n", analysis.Memory.P95)
		fmt.Printf("  P99:    %.1f%%\n", analysis.Memory.P99)
		fmt.Printf("  Max:    %.1f%%\n", analysis.Memory.Max)
		fmt.Println()
	}

	if len(analysis.Observations) > 0 {
		fmt.Println("Observations:")
		for _, obs := range analysis.Observations {
			fmt.Printf("  • %s\n", obs)
		}
		fmt.Println()
	}
}

func printRecommendation(rec *analyzer.InstanceRecommendation) {
	fmt.Println("=== INSTANCE SIZING RECOMMENDATION ===")
	fmt.Println()

	fmt.Println("Recommended Configuration:")
	fmt.Printf("  vCPUs:  %d cores\n", rec.RecommendedCPU)
	fmt.Printf("  Memory: %.1f GB\n", rec.RecommendedMemory)
	fmt.Println()

	if rec.EstimatedSavings > 0 {
		fmt.Printf("Estimated Savings: %.0f%%\n\n", rec.EstimatedSavings)
	}

	fmt.Println("Rationale:")
	for _, reason := range rec.Rationale {
		fmt.Printf("  • %s\n", reason)
	}
	fmt.Println()

	if len(rec.Warnings) > 0 {
		fmt.Println("⚠️  Warnings:")
		for _, warning := range rec.Warnings {
			fmt.Printf("  • %s\n", warning)
		}
		fmt.Println()
	}

	fmt.Println("Suggested Instance Types:")
	for provider, instances := range rec.CloudProviderSuggestions {
		if len(instances) > 0 {
			fmt.Printf("  %s: %v\n", provider, instances)
		}
	}
}
