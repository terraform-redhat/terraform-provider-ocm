package e2e

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	client "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/segmentio/ksuid"
	"github.com/terraform-redhat/terraform-provider-ocm/ci/helper"
	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	accountRolesFilesDir         = "account_roles_files"
	terraformProviderOCMFilesDir = "terraform_provider_ocm_files"
)

var (
	randSuffix          string
	providerTempDir     string
	accountRolesTempDir string
	operatorRolePrefix  string
	accountRolePrefix   string
	ocmEnvironment      string
	openshiftVersion    string
	tokenFilter         string
	gatewayFilter       string
	clusterName         string
	ctx                 context.Context
	TestID              = ksuid.New()
	connection          *client.Connection
	logger              logging.Logger
)

var URLAliases = map[string]string{
	"https://api.openshift.com":             "production",
	"https://api.stage.openshift.com":       "staging",
	"https://api.integration.openshift.com": "integration",
	"http://localhost:8000":                 "local",
	"http://localhost:9000":                 "local",
}
var args struct {
	tokenURL         string
	gatewayURL       string
	token            string
	clientID         string
	clientSecret     string
	openshiftVersion string
}

func init() {
	flag.StringVar(
		&args.tokenURL,
		"token-url",
		"https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token",
		"Token URL.",
	)
	flag.StringVar(
		&args.gatewayURL,
		"gateway-url",
		"http://localhost:8001",
		"Gateway URL.",
	)
	flag.StringVar(
		&args.clientID,
		"client-id",
		"cloud-services",
		"OpenID client identifier.",
	)
	flag.StringVar(
		&args.clientSecret,
		"client-secret",
		"",
		"OpenID client secret.",
	)
	flag.StringVar(
		&args.token,
		"token",
		"",
		"Offline token for authentication.",
	)
	flag.StringVar(
		&args.openshiftVersion,
		"openshift-version",
		"4.12",
		"Openshift version.",
	)
}

func TestE2E(t *testing.T) {
	// Create the context:
	ctx = context.Background()
	// logger is the global logger used by the tests.
	logger = helper.GetLogger()

	// Parse the command line flags:
	flag.Parse()

	// Run the tests:
	logger.Info(ctx, "Test ID: %s", TestID)
	validateArgs()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Full Cycle Test Suite")

}

func validateArgs() {
	helper.CheckEmpty(args.gatewayURL, "gateway-url")
	helper.CheckEmpty(args.clientID, "client-id")
	helper.CheckEmpty(args.token, "token")
	helper.CheckEmpty(args.openshiftVersion, "openshift-version")
}

var _ = BeforeEach(func() {
	var err error

	// Replace the global logger with one specific for this test that writes to the Ginkgo
	// streams, that way the log messages will only be displayed if the test fails:
	logger, err = logging.NewStdLoggerBuilder().
		Streams(GinkgoWriter, GinkgoWriter).
		Build()
	Expect(err).ToNot(HaveOccurred())
})

var _ = BeforeSuite(func() {
	connection = helper.CreateConnectionWithToken(
		args.token,
		args.tokenURL,
		args.gatewayURL,
		args.clientID,
		args.clientSecret)
	helper.WaitForBackendToBeReady(ctx, connection)
})

func runTerraformInit(ctx context.Context, dir string) {
	logger.Info(ctx, "Running terraform init against the dir %s", dir)
	terraformInitCmd := exec.Command("terraform", "init")
	terraformInitCmd.Dir = dir
	err := terraformInitCmd.Run()
	helper.CheckError(err)
}

func runTerraformApplyWithArgs(ctx context.Context, dir string, terraformArgs []string) {
	applyArgs := append([]string{"apply"}, terraformArgs...)
	logger.Info(ctx, "Running terraform apply against the dir: %s", dir)
	terraformApply := exec.Command("terraform", applyArgs...)
	terraformApply.Dir = dir
	terraformApply.Stdout = os.Stdout
	terraformApply.Stderr = os.Stderr
	err := terraformApply.Run()
	helper.CheckError(err)
}
func runTerraformDestroyWithArgs(ctx context.Context, dir string, terraformArgs []string) {
	destroyArgs := append([]string{"destroy"}, terraformArgs...)
	logger.Info(ctx, "Running terraform destroy against the dir: %s", dir)
	terraformApply := exec.Command("terraform", destroyArgs...)
	terraformApply.Dir = dir
	terraformApply.Stdout = os.Stdout
	terraformApply.Stderr = os.Stderr
	err := terraformApply.Run()
	helper.CheckError(err)
}

func createAccountRolesUsingTerraformAWSRosaStsModule(ctx context.Context) {
	runTerraformInit(ctx, accountRolesTempDir)

	runTerraformApplyWithArgs(ctx, accountRolesTempDir, []string{
		"-var", accountRolePrefix,
		"-var", ocmEnvironment,
		"-var", openshiftVersion,
		"-auto-approve"})
}

func createClusterUsingTerraformProviderOCM(ctx context.Context) string {
	runTerraformInit(ctx, providerTempDir)

	runTerraformApplyWithArgs(ctx, providerTempDir, []string{
		"-var", tokenFilter,
		"-var", operatorRolePrefix,
		"-var", accountRolePrefix,
		"-var", gatewayFilter,
		"-var", clusterName,
		"-auto-approve"})

	getClusterIdCmd := exec.Command("terraform", "output", "-json", "cluster_id")
	getClusterIdCmd.Dir = providerTempDir
	output, err := getClusterIdCmd.Output()
	helper.CheckError(err)

	splitOutput := strings.Split(string(output), "\"")
	Expect(len(splitOutput)).To(BeNumerically(">", 1))

	return splitOutput[1]
}

func defineVariablesValues() {
	randSuffix = rand.String(4)
	logger.Info(ctx, "The random suffix that was chosen is %s", randSuffix)

	providerTempDir = fmt.Sprintf("%s_%s", terraformProviderOCMFilesDir, randSuffix)
	logger.Info(ctx, "The temp directory that was chosen is %s", providerTempDir)

	accountRolesTempDir = fmt.Sprintf("%s_%s", accountRolesFilesDir, randSuffix)
	logger.Info(ctx, "The temp directory that was chosen is %s", accountRolesTempDir)

	clusterName = fmt.Sprintf("cluster_name=terr-ocm-%s", randSuffix)
	logger.Info(ctx, "The cluster name that was chosen is %s", clusterName)

	operatorRolePrefix = fmt.Sprintf("operator_role_prefix=terr-operator-%s", randSuffix)
	logger.Info(ctx, "The operator IAM role prefix that was chose is %s", operatorRolePrefix)

	accountRolePrefix = fmt.Sprintf("account_role_prefix=terr-account-%s", randSuffix)
	logger.Info(ctx, "The account IAM role prefix that was chose is %s", accountRolePrefix)

	ocmEnvironment = fmt.Sprintf("ocm_env=%s", URLAliases[args.gatewayURL])
	logger.Info(ctx, "The ocm environment that was chose is %s", ocmEnvironment)

	openshiftVersion = fmt.Sprintf("rosa_openshift_version=%s", args.openshiftVersion)
	logger.Info(ctx, "The cluster version that was chose is %s", openshiftVersion)

	tokenFilter = fmt.Sprintf("token=%s", args.token)

	gatewayFilter = fmt.Sprintf("url=%s", args.gatewayURL)
	logger.Info(ctx, "The gateway url filter is %s", gatewayFilter)

}

var _ = FDescribe("Terraform provider OCM test", Ordered, func() {
	var terraformProviderOCMClusterID string

	BeforeAll(func() {
		defineVariablesValues()

		//create temp dirs
		err := Unpack(providerTempDir, terraformProviderOCMFilesDir)
		helper.CheckError(err)

		err = Unpack(accountRolesTempDir, accountRolesFilesDir)
		helper.CheckError(err)

		// prepareDirectory
		createAccountRolesUsingTerraformAWSRosaStsModule(ctx)
		time.Sleep(5 * time.Second)
		terraformProviderOCMClusterID = createClusterUsingTerraformProviderOCM(ctx)
	})

	Context("Cluster creation", func() {
		It("creates a cluster using terraform-provider-ocm", func() {
			resp, err := connection.ClustersMgmt().V1().Clusters().Cluster(terraformProviderOCMClusterID).Get().Send()
			helper.CheckResponse(resp, err, http.StatusOK)
		})
	})

	AfterAll(func() {
		// destroy cluster
		runTerraformDestroyWithArgs(ctx, providerTempDir, []string{
			"-var", tokenFilter,
			"-var", operatorRolePrefix,
			"-var", accountRolePrefix,
			"-var", gatewayFilter,
			"-var", clusterName,
			"-auto-approve"})

		// destroy account roles
		runTerraformDestroyWithArgs(ctx, accountRolesTempDir, []string{
			"-var", accountRolePrefix,
			"-var", ocmEnvironment,
			"-var", openshiftVersion,
			"-auto-approve"})

		// remove temporary directories
		os.RemoveAll(providerTempDir)
		os.RemoveAll(accountRolesTempDir)
	})
})

// Unpack unpacks the terraform tf files from this package into a target directory.
func Unpack(targetDir string, sourceDir string) (err error) {
	dir := ""
	Assets := http.Dir(dir)

	file, err := Assets.Open(sourceDir)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	if info.IsDir() {
		os.Mkdir(targetDir, 0777)
		children, err := file.Readdir(0)
		if err != nil {
			return err
		}
		file.Close()

		for _, childInfo := range children {
			name := childInfo.Name()
			err = Unpack(filepath.Join(targetDir, name), path.Join(sourceDir, name))
			if err != nil {
				return err
			}
		}
		return nil
	}

	out, err := os.OpenFile(targetDir, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	return err
}
