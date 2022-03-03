package webhook

import (
	server "github.com/alibaba/open-simulator/pkg/webhook"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var options server.Options

var WebhookServerCmd = &cobra.Command{
	Use:   "start",
	Short: "we need to start a webhook server in order to intervene scheduling of the relative pod by mutating pod in real cluster",
	Run: func(cmd *cobra.Command, args []string) {
		server.Start(options)
	},
}

func init() {
	WebhookServerCmd.Flags().IntVar(&options.Port, "port", 443, "")
	WebhookServerCmd.Flags().StringVar(&options.WebHookServerName, "server", "", "")
	WebhookServerCmd.Flags().StringVar(&options.WebHookNamespace, "namespace", "default", "")
	WebhookServerCmd.Flags().StringVar(&options.KubeConfig, "kube-config", "", "")
	//WebhookServerCmd.Flags().StringVar(&options.TLSCertFile, "tlsCertFile", "", "")
	//WebhookServerCmd.Flags().StringVar(&options.TLSKeyFile, "tlsKeyFile", "", "")
	WebhookServerCmd.Flags().StringVar(&options.MigrationPlanCfgFile, "migrationPlanCfgFile", "", "")

	//if err := WebhookServerCmd.MarkFlagRequired("tlsCertFile"); err != nil {
	//	log.Errorf("")
	//}
	//if err := WebhookServerCmd.MarkFlagRequired("tlsKeyFile"); err != nil {
	//	log.Errorf("")
	//}
	if err := WebhookServerCmd.MarkFlagRequired("migrationPlanCfgFile"); err != nil {
		log.Errorf("")
	}
}
