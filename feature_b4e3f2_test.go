// Code moved from command_test.go to make benchmark-compliant test file.
package cobra

import (
    "bytes"
    "strings"
    "testing"
)

func TestShipdExcludePersistentFlagPreventsInheritedFlags(t *testing.T) {
    rootCmd := &Command{Use: "root", Run: emptyRun}
    parentCmd := &Command{Use: "parent", Run: emptyRun}
    childCmd := &Command{Use: "child", Run: emptyRun}
    rootCmd.AddCommand(parentCmd)
    parentCmd.AddCommand(childCmd)

    rootCmd.PersistentFlags().Bool("rootflag", false, "")
    parentCmd.PersistentFlags().Bool("parentflag", false, "")

    childCmd.ExcludePersistentFlag("rootflag")

    if childCmd.InheritedFlags().Lookup("rootflag") != nil {
        t.Errorf("InheritedFlags should not contain excluded flag 'rootflag'")
    }
    if childCmd.Flags().Lookup("rootflag") != nil {
        t.Errorf("Flags should not contain excluded flag 'rootflag'")
    }
    if childCmd.InheritedFlags().Lookup("parentflag") == nil {
        t.Errorf("InheritedFlags expected to contain parent persistent flag 'parentflag'")
    }

    if childCmd.ExcludedPersistentFlags()[0] != "rootflag" {
        t.Errorf("ExcludedPersistentFlags expected [rootflag], got %v", childCmd.ExcludedPersistentFlags())
    }
}

func TestShipdClearExcludedPersistentFlagRestoresInheritedFlags(t *testing.T) {
    rootCmd := &Command{Use: "root", Run: emptyRun}
    parentCmd := &Command{Use: "parent", Run: emptyRun}
    childCmd := &Command{Use: "child", Run: emptyRun}
    rootCmd.AddCommand(parentCmd)
    parentCmd.AddCommand(childCmd)

    rootCmd.PersistentFlags().Bool("rootflag", false, "")

    childCmd.ExcludePersistentFlag("rootflag")
    childCmd.ClearExcludedPersistentFlag()

    if childCmd.InheritedFlags().Lookup("rootflag") == nil {
        t.Errorf("InheritedFlags expected to contain restored flag 'rootflag'")
    }
    if len(childCmd.ExcludedPersistentFlags()) != 0 {
        t.Errorf("ExcludedPersistentFlags expected empty after clear, got %v", childCmd.ExcludedPersistentFlags())
    }
}

func TestShipdExcludePersistentFlagFiltersDescendants(t *testing.T) {
    rootCmd := &Command{Use: "root", Run: emptyRun}
    parentCmd := &Command{Use: "parent", Run: emptyRun}
    childCmd := &Command{Use: "child", Run: emptyRun}
    grandChildCmd := &Command{Use: "grandchild", Run: emptyRun}
    rootCmd.AddCommand(parentCmd)
    parentCmd.AddCommand(childCmd)
    childCmd.AddCommand(grandChildCmd)

    rootCmd.PersistentFlags().Bool("rootflag", false, "")

    childCmd.ExcludePersistentFlag("rootflag")

    if grandChildCmd.InheritedFlags().Lookup("rootflag") != nil {
        t.Errorf("InheritedFlags should not contain excluded flag 'rootflag' in descendant")
    }
}

func TestShipdExcludePersistentFlagUpdatesAfterParentFlagAdded(t *testing.T) {
    rootCmd := &Command{Use: "root", Run: emptyRun}
    childCmd := &Command{Use: "child", Run: emptyRun}
    rootCmd.AddCommand(childCmd)

    childCmd.ExcludePersistentFlag("rootflag")
    rootCmd.PersistentFlags().Bool("rootflag", false, "")
    rootCmd.PersistentFlags().Bool("otherflag", false, "")

    if childCmd.InheritedFlags().Lookup("rootflag") != nil {
        t.Errorf("InheritedFlags should not contain excluded flag 'rootflag' after parent flag added")
    }
    if childCmd.InheritedFlags().Lookup("otherflag") == nil {
        t.Errorf("InheritedFlags expected to contain parent flag 'otherflag' after parent flag added")
    }
}

func TestShipdExcludePersistentFlagIntegration_ExecuteCAndHelp(t *testing.T) {
    buildRoot := func() (*Command, *Command, *Command, *Command, *bool, *bool) {
        rootCmd := &Command{Use: "root", TraverseChildren: true, Run: emptyRun}
        childA := &Command{Use: "childA", Run: emptyRun}
        grandchildA := &Command{Use: "grandchildA", Run: emptyRun}
        childB := &Command{Use: "childB", Run: emptyRun}
        rootCmd.AddCommand(childA, childB)
        childA.AddCommand(grandchildA)

        var rootFlagValue bool
        var rootFlag2Value bool
        rootCmd.PersistentFlags().BoolVar(&rootFlagValue, "rootflag", false, "root flag")
        rootCmd.PersistentFlags().BoolVar(&rootFlag2Value, "rootflag2", false, "root flag 2")

        childA.ExcludePersistentFlag("rootflag")

        return rootCmd, childA, childB, grandchildA, &rootFlagValue, &rootFlag2Value
    }

    cases := []struct {
        name              string
        args              []string
        wantErr           bool
        wantErrorContains string
        wantCmd           string
        wantRootFlag      bool
        wantRootFlag2     bool
    }{
        {
            name:          "accepted nonexcluded persistent flag in excluded subtree",
            args:          []string{"childA", "--rootflag2"},
            wantErr:       false,
            wantCmd:       "childA",
            wantRootFlag:  false,
            wantRootFlag2: true,
        },
        {
            name:              "rejected excluded persistent flag in excluded subtree after subcommand",
            args:              []string{"childA", "--rootflag"},
            wantErr:           true,
            wantErrorContains: "unknown flag",
        },
        {
            name:              "rejected excluded persistent flag in excluded descendant subtree",
            args:              []string{"childA", "grandchildA", "--rootflag"},
            wantErr:           true,
            wantErrorContains: "unknown flag",
        },
        {
            name:          "accepted root flag in unaffiliated sibling subtree",
            args:          []string{"childB", "--rootflag"},
            wantErr:       false,
            wantCmd:       "childB",
            wantRootFlag:  true,
            wantRootFlag2: false,
        },
        {
            name:          "accepted root flag when specified before excluded subcommand",
            args:          []string{"--rootflag", "childA"},
            wantErr:       false,
            wantCmd:       "childA",
            wantRootFlag:  true,
            wantRootFlag2: false,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            rootCmd, _, _, _, rootFlagValue, rootFlag2Value := buildRoot()
            *rootFlagValue = false
            *rootFlag2Value = false

            cmd, output, err := executeCommandC(rootCmd, tc.args...)
            if tc.wantErr {
                if err == nil {
                    t.Fatalf("expected an error for args %v", tc.args)
                }
                if !strings.Contains(output, tc.wantErrorContains) && !strings.Contains(err.Error(), tc.wantErrorContains) {
                    t.Fatalf("expected output or error to contain %q, got output=%q err=%v", tc.wantErrorContains, output, err)
                }
                return
            }

            if err != nil {
                t.Fatalf("unexpected error for args %v: %v, output=%q", tc.args, err, output)
            }
            if tc.wantCmd != "" && cmd.Name() != tc.wantCmd {
                t.Fatalf("expected command %q, got %q", tc.wantCmd, cmd.Name())
            }
            if *rootFlagValue != tc.wantRootFlag {
                t.Fatalf("unexpected rootflag value; want %v, got %v", tc.wantRootFlag, *rootFlagValue)
            }
            if *rootFlag2Value != tc.wantRootFlag2 {
                t.Fatalf("unexpected rootflag2 value; want %v, got %v", tc.wantRootFlag2, *rootFlag2Value)
            }
        })
    }

    helpCases := []struct {
        name     string
        args     []string
        contains []string
        omits    []string
    }{
        {
            name:     "root help includes both persistent flags",
            args:     []string{"--help"},
            contains: []string{"--rootflag ", "--rootflag2"},
        },
        {
            name:     "childA help omits excluded root flag but includes allowed root flag",
            args:     []string{"childA", "--help"},
            contains: []string{"--rootflag2"},
            omits:    []string{"--rootflag "},
        },
        {
            name:     "grandchildA help omits excluded root flag but includes allowed root flag",
            args:     []string{"childA", "grandchildA", "--help"},
            contains: []string{"--rootflag2"},
            omits:    []string{"--rootflag "},
        },
        {
            name:     "childB help includes both persistent flags",
            args:     []string{"childB", "--help"},
            contains: []string{"--rootflag ", "--rootflag2"},
        },
    }

    for _, tc := range helpCases {
        t.Run(tc.name, func(t *testing.T) {
            rootCmd, _, _, _, _, _ := buildRoot()
            output, err := executeCommand(rootCmd, tc.args...)
            if err != nil {
                t.Fatalf("unexpected error for help args %v: %v", tc.args, err)
            }
            for _, want := range tc.contains {
                checkFlagContains(t, output, want)
            }
            for _, omit := range tc.omits {
                checkFlagOmits(t, output, omit)
            }
        })
    }
}

func TestShipdExcludePersistentFlagIntegration_Completion(t *testing.T) {
    buildRoot := func() (*Command, *Command, *Command) {
        rootCmd := &Command{Use: "root", TraverseChildren: true, Run: emptyRun}
        childA := &Command{Use: "childA", Run: emptyRun}
        childB := &Command{Use: "childB", Run: emptyRun}
        rootCmd.AddCommand(childA, childB)

        rootCmd.PersistentFlags().Bool("rootflag", false, "root flag")
        rootCmd.PersistentFlags().Bool("rootflag2", false, "root flag 2")
        childA.ExcludePersistentFlag("rootflag")

        childA.Flags().Bool("childaflag", false, "child a flag")
        childB.Flags().Bool("childbflag", false, "child b flag")

        return rootCmd, childA, childB
    }

    t.Run("childA completion omits excluded root flag", func(t *testing.T) {
        rootCmd, _, _ := buildRoot()
        output, err := executeCommand(rootCmd, ShellCompRequestCmd, "childA", "-")
        if err != nil {
            t.Fatalf("unexpected error during completion: %v", err)
        }
        checkFlagContains(t, output, "--rootflag2")
        checkFlagOmits(t, output, "--rootflag")
    })

    t.Run("childB completion includes excluded root flag", func(t *testing.T) {
        rootCmd, _, _ := buildRoot()
        output, err := executeCommand(rootCmd, ShellCompRequestCmd, "childB", "-")
        if err != nil {
            t.Fatalf("unexpected error during completion: %v", err)
        }
        checkFlagContains(t, output, "--rootflag")
        checkFlagContains(t, output, "--rootflag2")
    })

    t.Run("bash completion script generates shell request command", func(t *testing.T) {
        rootCmd, _, _ := buildRoot()
        buf := new(bytes.Buffer)
        assertNoErr(t, rootCmd.GenBashCompletion(buf))
        checkStringContains(t, buf.String(), ShellCompRequestCmd)
    })

    t.Run("zsh completion script generates shell request command", func(t *testing.T) {
        rootCmd, _, _ := buildRoot()
        buf := new(bytes.Buffer)
        assertNoErr(t, rootCmd.GenZshCompletion(buf))
        checkStringContains(t, buf.String(), ShellCompRequestCmd)
    })
}
