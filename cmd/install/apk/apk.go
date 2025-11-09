package apk

import (
	"fmt"
	"os"

	"github.com/devcontainer-community/nanolayer-go/internal/installers/apk"
	"github.com/spf13/cobra"
)

var ApkCmd = &cobra.Command{
	Use:   "apk [packages...]",
	Short: "Install packages using APK (Alpine Package Keeper)",
	Long:  `Install packages on Alpine Linux using the APK package manager.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Error: At least one package name is required.")
			os.Exit(1)
		}

		fmt.Printf("Installing packages: %v\n", args)

		err := apk.InstallPackage(args)
		if err != nil {
			fmt.Printf("Error during installation: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Installation completed successfully!")
	},
}

func init() {

}
