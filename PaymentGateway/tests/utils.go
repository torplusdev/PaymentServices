package tests

import (
	"context"
	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"log"
	"paidpiper.com/payment-gateway/common"
	"strconv"
	"strings"
	"testing"
)


// ##############     Test seeds     #################################################
const User1Seed = "SC33EAUSEMMVSN4L3BJFFR732JLASR4AQY7HBRGA6BVKAPJL5S4OZWLU"
const Service1Seed = "SBBNHWCWUFLM4YXTF36WUZP4A354S75BQGFGUMSAPCBTN645TERJAC34"

// public GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN
const Node1Seed = "SCEV4AU2G4NYAW76P46EVM77N5TL2NLW2IYO5TJSLB6S4OBBJQ62ZVJN"

// public GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ
const Node2Seed = "SDK7QBPKP5M7SCU7XZVWAIUJW2I2SM4PQJMWH5PSCMAI7WF3A4HRHVVC"

// public GB3IKDN72HFZSLY3SYE5YWULA5HG32AAKEDJTG6J6X2YKITHBDDT2PIW
const Node3Seed = "SBZMAHJPLZLDKJU4DUIT6AU3BEVWKPGP6M6L2KWZXAELKNAIDADGZO7A"

// publc GASFIR7LHA2IAAMLN4WMBKPSFL6GSQGWHF3E7PHHGFADT254PBOOY2I7
const Node4Seed = "SBVOHS5MWK5OHDFSCURZD7XZXTETKSRTKSFMU2IKJXUBM23I5FJHWDXK"

type Sampler interface {
	ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult
	Description() string
}

func InitGlobalTracer() (*sdktrace.Provider,func()) {

	// Create and install Jaeger export pipeline
	provider, flush, err := jaeger.NewExportPipeline(
		jaeger.WithCollectorEndpoint("http://192.168.162.128:14268/api/traces"),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: "tests",
			Tags: []core.KeyValue{
				key.String("exporter", "jaeger"),
			},
		}),
		/// jaeger.RegisterAsGlobal() creates a lot of noise because of net/http traces, use it only if you really have to

		//jaeger.WithSDK(&sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),


		// NeverSample disables sampling
		jaeger.WithSDK(&sdktrace.Config{DefaultSampler: sdktrace.NeverSample()}),
	)
	if err != nil {
		log.Fatal(err)
	}

	_ = flush

	//return provider, flush
	return provider, nil
}

func InitTestCreateSpan(t *testing.T, spanName string) (*assert.Assertions,context.Context, trace.Span) {

	asserter := assert.New(t)
	tr := common.CreateTracer("test")

	ctx := correlation.NewContext(context.Background(),
		key.String("test", spanName),
	)

	ctx,span := tr.Start(ctx,spanName)

	return asserter, ctx, span
}

func GetAccountBalances(seeds []string) []float64 {

	balances := make([]float64,len(seeds))

	for i,seed := range seeds {
		kp, _ := keypair.ParseFull(seed)
		acc, _ := GetAccount(kp.Address())

		strBalance := acc.GetCreditBalance(common.PPTokenAssetName, common.PPTokenIssuerAddress)

		balances[i], _ = strconv.ParseFloat(strBalance, 64)
	}

	return balances
}

func CreateAndFundAccount(seed string) {

	client := horizonclient.DefaultTestNetClient

	pair, err := keypair.ParseFull(seed)

	if err != nil {
		log.Fatal(err)
	}

	detail, errAccount := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: pair.Address()})

	if errAccount != nil {

		txSuccess, errCreate := client.Fund(pair.Address())

		if errCreate != nil {
			log.Fatal(err)
		}

		log.Printf("Account "+seed+" created - trans#:", txSuccess.Hash)
	}

	if detail.GetCreditBalance(common.PPTokenAssetName, common.PPTokenIssuerAddress) == "0" {

		distributionKp, err := keypair.ParseFull("SAQUH66AMZ3PURY2G3ROXRXGIF2JMZC7QFVED65PYP4YJQFIWCPCWKPM")
		if err != nil {
			log.Fatal(err)
		}

		issuerKp, err := keypair.ParseFull("SBMCAMFAYTXFIXBAOZJE5X2ZX4TJQI5X6P6NE5SHOEBHLHEMGKANRTOQ")
		if err != nil {
			log.Fatal(err)
		}

		distributionAccountDetail, err := client.AccountDetail(
			horizonclient.AccountRequest{
				AccountID: distributionKp.Address()})

		if err != nil {
			log.Fatal(err)
		}

		// Create trust line
		tokenAsset := txnbuild.CreditAsset{
			Code:   "pptoken",
			Issuer: issuerKp.Address(),
		}

		changeTrust := txnbuild.ChangeTrust{
			SourceAccount: &distributionAccountDetail,
			Line:          tokenAsset,
			Limit:         "100000",
		}

		txCreateTrustLine := txnbuild.Transaction{
			SourceAccount: &distributionAccountDetail,
			Operations:    []txnbuild.Operation{&changeTrust},
			Timebounds:    txnbuild.NewTimeout(300),
			Network:       network.TestNetworkPassphrase,
		}

		xdr, err := txCreateTrustLine.BuildSignEncode(distributionKp)

		_ = xdr
		if err != nil {
			log.Print("Error signing transaction:")
		}

		_, err = client.SubmitTransaction(txCreateTrustLine)

		if err != nil {
			log.Fatal(err)
		}

		strBalance := detail.GetCreditBalance(common.PPTokenAssetName, common.PPTokenIssuerAddress)
		balance, _ := strconv.ParseFloat(strBalance, 32)

		if balance < 1000 {
			err = injectFundsPPToken(pair)

		}
	}
}

func injectFundsPPToken(kp *keypair.Full) error {

	// Inject lumens, just in case
	err := injectFundsXLM(kp.Address())

	if err != nil {
		return err
	}

	client := horizonclient.DefaultTestNetClient
	pair, _ := keypair.Random()

	_, errCreate := client.Fund(pair.Address())

	if errCreate != nil { return errCreate}

	distributionKp,_ := keypair.ParseFull("SAQUH66AMZ3PURY2G3ROXRXGIF2JMZC7QFVED65PYP4YJQFIWCPCWKPM")

	accountDistribution, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: distributionKp.Address()})


	accountTarget, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: kp.Address()})


	_ = accountDistribution

	tokenAsset  := txnbuild.CreditAsset{
		Code:   "pptoken",
		Issuer: common.PPTokenIssuerAddress,
	}

	changeTrust := txnbuild.ChangeTrust{
		SourceAccount: &accountTarget,
		Line:tokenAsset,
		Limit:"2000",
	}

	txCreateTrustLine := txnbuild.Transaction{
		SourceAccount: &accountTarget,
		Operations:    []txnbuild.Operation{ &changeTrust},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	_, err = txCreateTrustLine.BuildSignEncode(kp)
	resp, err := client.SubmitTransaction(txCreateTrustLine)

	if err != nil {
		log.Fatal(err)
		return err
	}

	distributeAssets := txnbuild.Payment{
		Destination:   kp.Address(),
		Amount:        "1000",
		Asset:         tokenAsset,
		SourceAccount: &accountDistribution,
	}

	txDistributeAssets := txnbuild.Transaction{
		SourceAccount: &accountDistribution,
		Operations:    []txnbuild.Operation{ &distributeAssets},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	_, err = txDistributeAssets.BuildSignEncode(distributionKp)

	resp, err = client.SubmitTransaction(txDistributeAssets)

	if err != nil {
		log.Fatal(err)
		return err
	}

	_ = resp

	return nil
}

func injectFundsXLM(address string) error {

	client := horizonclient.DefaultTestNetClient
	pair, _ := keypair.Random()

	_, errCreate := client.Fund(pair.Address())

	if errCreate != nil { return errCreate}

	account, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: pair.Address()})

	currentBalance,_ :=account.GetNativeBalance()
	_ = currentBalance

	amount := 9900

	tx, err := build.Transaction(
		build.TestNetwork,
		build.SourceAccount{pair.Seed()},
		build.AutoSequence{horizon.DefaultTestNetClient},
		build.Payment(
			build.Destination{address},
			build.NativeAmount{strconv.Itoa(amount)},
		),
	)

	if err != nil { return err}

	txe, err := tx.Sign(pair.Seed())

	txeB64, err := txe.Base64()

	resp, err := horizon.DefaultTestNetClient.SubmitTransaction(txeB64)

	_ = resp.Hash

	return nil
}

func SetSigners(seed string, signerSeed string) {

	client := horizonclient.DefaultTestNetClient

	pair, err := keypair.ParseFull(seed)

	if err != nil {
		log.Fatal(err)
	}

	signerPair, err := keypair.ParseFull(signerSeed)
	if err != nil { log.Fatal(err) 	}

	clientAccount := txnbuild.NewSimpleAccount(pair.Address(),0)

	setOptionsChangeWeights := txnbuild.SetOptions{
		SourceAccount: &clientAccount,
		Signer: &txnbuild.Signer{
			Address: signerPair.Address(),
			Weight:  10,
		},
	}

	tx := txnbuild.Transaction{
		SourceAccount: &clientAccount,
		Operations:    []txnbuild.Operation{ &setOptionsChangeWeights},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	tx.Build()
	tx.Sign(pair)

	resp, err := client.SubmitTransaction(tx)
	if err != nil {
		hError := err.(*horizonclient.Error)
		log.Fatal("Error submitting transaction:", hError,hError.Problem)
	}

	_ = resp
}
func GetAccount(address string) (account horizon.Account, err error) {

	client := horizonclient.DefaultTestNetClient

	accountDetail, errAccount := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: address})

	return accountDetail, errAccount
}

func Print(t *common.PaymentTransaction) string {
	b := strings.Builder{}

	b.WriteString("ext.src: " + t.TransactionSourceAddress + "\n")
	b.WriteString("ext.adr: " + t.PaymentDestinationAddress + "\n")

	internalTrans, err := txnbuild.TransactionFromXDR(t.XDR)

	if err != nil {
		return "Err: " + err.Error()
	}

	b.WriteString("trans.srcAccount: " + internalTrans.SourceAccount.GetAccountID() + "\n")

	for _, signature := range internalTrans.TxEnvelope().Signatures {
		b.WriteString("Signature [" + strconv.Itoa(signature.Signature.XDRMaxSize()) + "]")
	}

	for _, op := range internalTrans.Operations {
		xdrOp, _ := op.BuildXDR()
		b.WriteString("trans.op <" + xdrOp.Body.Type.String() + ">" + "\n")

		switch xdrOp.Body.Type {
		case xdr.OperationTypePayment:
			payment := &txnbuild.Payment{}

			err = payment.FromXDR(xdrOp)

			if err != nil {
				return "Error converting from XDR: " + err.Error()
			}

			b.WriteString("    from:" + payment.SourceAccount.GetAccountID() + "\n")
			b.WriteString("      to:" + payment.Destination + "\n")
			b.WriteString("  amount:" + payment.Amount + "\n")

		default:
			return "Unexpected operation type: " + xdrOp.Body.Type.String()
		}
	}

	return b.String()
}
