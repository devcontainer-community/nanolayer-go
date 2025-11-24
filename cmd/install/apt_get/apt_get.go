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

		err_1 := aptget.AddRepositoryKey("https://pkg.cloudflareclient.com/pubkey.gpg", "/usr/share/keyrings/cloudflare-warp-archive-keyring.gpg", true)
		if err_1 != nil {
			fmt.Printf("Error adding repository key: %v\n", err_1)
			os.Exit(1)
		}

		err_2 := aptget.AddAptRepository("https://pkg.cloudflareclient.com/", "/usr/share/keyrings/cloudflare-warp-archive-keyring.gpg", "jammy", "main", "/etc/apt/sources.list.d/cloudflare-client.list")
		if err_2 != nil {
			fmt.Printf("Error adding apt repository: %v\n", err_2)
			os.Exit(1)
		}

		err_3 := aptget.UpdatePackageLists()
		if err_3 != nil {
			fmt.Printf("Error updating package lists: %v\n", err_3)
			os.Exit(1)
		}

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
