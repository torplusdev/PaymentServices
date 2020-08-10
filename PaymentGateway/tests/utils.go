package tests

import (
	"context"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"paidpiper.com/payment-gateway/common"
)

// ##############     Test seeds     #################################################
//GCR5Q6Y25VBRX4EDOH2IYEQFQ5BNJPWN5DGNYEWJ5PXH7IVJPIFAVMGO
const User1Seed = "SBBHR4B4W4ENUXY23GQ2LBFJTOBRJLGRJYWEBQXOEUFTMN4NF4AHI2TD"

//GBG6HV7GTZ2QWIERNEFYTSVEAXRINYJTKM3ZGWIMVXWD5II5TNMHJW2V
const Service1Seed = "SBVKDDNEPATDCDAHEZZXWX6FA6MDOYIUJFKLNBEITGTVOQKNI2PDYCUO"

// public GBDCHSBQ5ATYKLKR65AV3FNSWAJXUFT6EFIY2TEAJYAFG3NEEOH3PUQ6
const Node1Seed = "SBY4B6ZQLH3NNDBRTHH6KHRS4MRRPEBOKFNSJSCUOMYVM4K734FTDUMG"

// public GB4EMKPLFADFZGHSDQDLILGKX5T3VOMBG444ZXBOZCXN2VXVDFP7PH27
const Node2Seed = "SBYKNN2PGYZNFEIZBSQJX6SXGQJ2MIZE5H7H54Z4OUBN7IFOLLKN5VH7"

// public GBJP7NUQON35NK3AL2ECG64QIP6Y6D32NNZLQJDJ5NMDNVPXOOTVG72I
const Node3Seed = "SCPAR4SZ225QLWP73VPNY5H3QEPXKFVFK6FDPC2EPG7DMLV3CZCZNVLF"

// publc GCVNUZK4YIXYIH6GTOL42HR43O422CPCSGRURTCXVST77CRKIOV7QHQO
const Node4Seed = "SBGW75JAGFA3JWH2MJEKLSUYMMXO42LKYBOIWTSI727MU7VH7FOOJLOZ"

type NodeRoleType string

const(
	Client NodeRoleType = "client"
	Node = "service_node"
)

type Sampler interface {
	ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult
	Description() string
}

func InitGlobalTracer() (*sdktrace.Provider, func()) {

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

func InitTestCreateSpan(t *testing.T, spanName string) (*assert.Assertions, context.Context, trace.Span) {

	asserter := assert.New(t)
	tr := common.CreateTracer("test")

	ctx := correlation.NewContext(context.Background(),
		key.String("test", spanName),
	)

	ctx, span := tr.Start(ctx, spanName)

	return asserter, ctx, span
}

func GetAccountBalances(seeds []string) []float64 {

	balances := make([]float64, len(seeds))

	for i, seed := range seeds {
		kp, _ := keypair.ParseFull(seed)
		acc, _ := GetAccount(kp.Address())

		strBalance := acc.GetCreditBalance(common.PPTokenAssetName, common.PPTokenIssuerAddress)

		balances[i], _ = strconv.ParseFloat(strBalance, 64)
	}

	return balances
}

func CreateAndFundAccount(seed string, role NodeRoleType) {

	client := horizonclient.DefaultTestNetClient

	pair, err := keypair.ParseFull(seed)

	// TODO: Move this to somewhere central
	kpManager,_ := keypair.ParseFull("SAT3ZXAC5IQHF753DLROYVW5HRZGGFB2BHEXDWMDHCHE2URPSSDW3NY5")

	if err != nil {
		log.Fatal(err)
	}

	detail, errAccount := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: pair.Address()})

	// Check that account exists
	if errAccount != nil {
		//TODO: call creation logic
		log.Fatal ("Account doesn't exist")
		//txSuccess, errCreate := client.Fund(pair.Address())
		//
		//if errCreate != nil {
		//	log.Fatal(err)
		//}
		//
		//log.Printf("Account "+seed+" created - trans#:", txSuccess.Hash)
	}

	var weight byte

	for _,signer := range detail.Signers {
		if signer.Key == pair.Address() {
			weight = byte(signer.Weight)
		}
	}

	if ((role == Client) && (weight < detail.Thresholds.MedThreshold)) ||
		((role == Node) && (weight < detail.Thresholds.MedThreshold)) {

		targetWeight := byte(detail.Thresholds.MedThreshold)

		//if (role == Client) {
		//	targetWeight = 0
		//}

		threshold := txnbuild.Threshold(targetWeight)

		setOptions := txnbuild.SetOptions{
			MasterWeight: &threshold ,
			//Signer: &txnbuild.Signer{
			//	Address: pair.Address(),
			//	Weight:  txnbuild.Threshold(targetWeight),
			//},
			SourceAccount: &detail,
		}

		txChangeSignature,err := txnbuild.NewTransaction(txnbuild.TransactionParams{
			SourceAccount:        &detail,
			IncrementSequenceNum: true,
			Operations:           []txnbuild.Operation{&setOptions},
			BaseFee:              300,
			Timebounds:           txnbuild.NewTimeout(300),
		})

		_ = err

		signedTransactionManager, err := txChangeSignature.Sign(network.TestNetworkPassphrase,kpManager)
		//signedTransactionManagerClient, err := signedTransactionManager.Sign(network.TestNetworkPassphrase,pair)
		_, err = client.SubmitTransaction(signedTransactionManager)

		if err != nil {
			log.Fatal("Can't change signature permissions: " + err.Error())
		}

	}

	hasBalance := false
	for _,b := range detail.Balances {
		if b.Issuer == common.PPTokenIssuerAddress && b.Code == common.PPTokenAssetName {
			hasBalance = true;
		}
	}

	hasBalance = false
	if (!hasBalance) {

		// Create trust line
		tokenAsset := txnbuild.CreditAsset{
			Code:   common.PPTokenAssetName,
			Issuer: common.PPTokenIssuerAddress,
		}

		changeTrust := txnbuild.ChangeTrust{
			SourceAccount: &detail,
			Line:          tokenAsset,
			Limit:         strconv.Itoa(1000),
		}

		txCreateTrustLine,err := txnbuild.NewTransaction(txnbuild.TransactionParams{
			SourceAccount:        &detail,
			IncrementSequenceNum: true,
			BaseFee: 200,
			Operations:           []txnbuild.Operation{&changeTrust},
			Timebounds:           txnbuild.NewTimeout(300),
		})

		signedTransaction, err := txCreateTrustLine.Sign(network.TestNetworkPassphrase,kpManager)

		_, err = client.SubmitTransaction(signedTransaction)

		if err != nil {
			log.Print("Error:" + err.Error())
		}
	}

	strBalance := detail.GetCreditBalance(common.PPTokenAssetName, common.PPTokenIssuerAddress)
	balance, _ := strconv.ParseFloat(strBalance, 32)

	if balance < 200 {
		err = injectFundsPPToken(pair, int(299-balance))
		if err != nil {
			log.Print("Error injecting pptoken funds: " + err.Error())
		}
	}
}

func injectFundsPPToken(kp *keypair.Full, amount int) error {

	// Inject lumens, just in case
	err := injectFundsXLM(kp.Address())

	if err != nil {
		return err
	}


	strAmount := strconv.Itoa(amount)

	client := horizonclient.DefaultTestNetClient
	pair, _ := keypair.Random()

	_, errCreate := client.Fund(pair.Address())

	if errCreate != nil {
		return errCreate
	}

	distributionKp, _ := keypair.ParseFull("SAQUH66AMZ3PURY2G3ROXRXGIF2JMZC7QFVED65PYP4YJQFIWCPCWKPM")
	issuerKp, _ := keypair.ParseFull("SBMCAMFAYTXFIXBAOZJE5X2ZX4TJQI5X6P6NE5SHOEBHLHEMGKANRTOQ")

	accountDistribution, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: distributionKp.Address()})

	accountIssuer, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: issuerKp.Address()})

	accountTarget, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: kp.Address()})

	_ = accountTarget
	_ = accountDistribution
	_ = accountIssuer

	tokenAsset := txnbuild.CreditAsset{
		Code:   "pptoken",
		Issuer: common.PPTokenIssuerAddress,
	}

	hasPPTokenBalance := false

	for _,b := range accountTarget.Balances {
		if b.Asset.Code == common.PPTokenAssetName {
			hasPPTokenBalance = true
		}
	}

	if (!hasPPTokenBalance) {

		changeTrust := txnbuild.ChangeTrust{
			SourceAccount: &accountIssuer,
			Line:          tokenAsset,
		}

		txCreateTrustLine, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
			SourceAccount:  &accountTarget,
			IncrementSequenceNum: true,
			Operations:    	[]txnbuild.Operation{&changeTrust},
			BaseFee: 200,
			Timebounds:     txnbuild.NewTimeout(300),
		})

		if err != nil {
			log.Print("Error creating transaction:")
		}

		txCreateTrustLineSignedBoth, err := txCreateTrustLine.Sign(network.TestNetworkPassphrase, kp)
		_, err = client.SubmitTransaction(txCreateTrustLineSignedBoth)

		if err != nil {
			log.Fatal(err)
			return err
		}
	}

	distributeAssets := txnbuild.Payment{
		Destination:   kp.Address(),
		Amount:        strAmount,
		Asset:         tokenAsset,
		SourceAccount: &accountDistribution,
	}

	txDistributeAssets, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &accountDistribution,
		IncrementSequenceNum: true,
		Operations:    []txnbuild.Operation{&distributeAssets},
		Timebounds:    txnbuild.NewTimeout(300),
		BaseFee: 200,
	})

	txDistributeAssetsSigned, err := txDistributeAssets.Sign(network.TestNetworkPassphrase, distributionKp)

	_, err = client.SubmitTransaction(txDistributeAssetsSigned)

	if err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

func injectFundsXLM(address string) error {

	client := horizonclient.DefaultTestNetClient
	pair, _ := keypair.Random()

	_, errCreate := client.Fund(pair.Address())

	if errCreate != nil {
		return errCreate
	}

	account, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: pair.Address()})

	currentBalance, _ := account.GetNativeBalance()
	_ = currentBalance

	amount := 9900

	sourceAccount, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: pair.Address()})

	payment := txnbuild.Payment{
		Destination:   address,
		Amount:        strconv.Itoa(amount),
		Asset: txnbuild.NativeAsset{},
		SourceAccount: &sourceAccount,
	}

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &sourceAccount,
		IncrementSequenceNum: true,
		BaseFee: 200,
		Operations:           []txnbuild.Operation{&payment},
		Timebounds: 		  txnbuild.NewTimeout(300),
	})

	if err != nil {
		return err
	}

	txe, err := tx.Sign(network.TestNetworkPassphrase, pair)

	//txeB64, err := txe.Base64()

	txTrans,err := horizonclient.DefaultTestNetClient.SubmitTransaction(txe)

	_ = txTrans

	return nil
}

func SetSigners(seed string, signerSeed string) {

	client := horizonclient.DefaultTestNetClient

	pair, err := keypair.ParseFull(seed)

	if err != nil {
		log.Fatal(err)
	}

	signerPair, err := keypair.ParseFull(signerSeed)
	if err != nil {
		log.Fatal(err)
	}

	clientAccount := txnbuild.NewSimpleAccount(pair.Address(), 0)

	setOptionsChangeWeights := txnbuild.SetOptions{
		SourceAccount: &clientAccount,
		Signer: &txnbuild.Signer{
			Address: signerPair.Address(),
			Weight:  10,
		},
	}

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &clientAccount,
		Operations:    []txnbuild.Operation{&setOptionsChangeWeights},
		Timebounds:    txnbuild.NewTimeout(300),
		BaseFee: 200,
	})

	if err != nil {
		log.Print("Error creating transaction")
	}

	_,err = tx.Sign(network.TestNetworkPassphrase,pair)

	if err != nil {
		log.Print("Error signing transaction")
	}

	resp, err := client.SubmitTransaction(tx)
	if err != nil {
		hError := err.(*horizonclient.Error)
		log.Fatal("Error submitting transaction:", hError, hError.Problem)
	}

	_ = resp
}
func GetAccount(address string) (account hProtocol.Account, err error) {

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

	trans, res := internalTrans.Transaction()

	if !res {
		b.WriteString("Error unpacking transaction!")
		return b.String()
	}


	if err != nil {
		return "Err: " + err.Error()
	}

	b.WriteString("trans.srcAccount: " + trans.SourceAccount().AccountID + "\n")

	for _, signature := range trans.Signatures() {
		b.WriteString("Signature [" + strconv.Itoa(signature.Signature.XDRMaxSize()) + "]")
	}

	for _, op := range trans.Operations() {
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
