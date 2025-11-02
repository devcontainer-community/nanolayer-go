package aptget

import (
	"fmt"
	"os"

	aptget "github.com/devcontainer-community/nanolayer-go/internal/installers/apt_get"
	"github.com/spf13/cobra"
)

var AptCmd = &cobra.Command{
	Use:   "apt [packages...]",
	Short: "Install packages using apt-get (Debian-based systems)",
	Long:  `Install packages on Debian like Linux using the apt-get package manager.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Error: At least one package name is required.")
			os.Exit(1)
		}

		fmt.Printf("Installing packages: %v\n", args)

		err := aptget.InstallPackage(args)
		if err != nil {
			fmt.Printf("Error during installation: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Installation completed successfully!")
	},
}

func init() {

}
