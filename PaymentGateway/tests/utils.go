package testutils

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"log"
	"paidpiper.com/payment-gateway/common"
	"strconv"
	"strings"
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

func InitGlobalTracer() func() {

	// Create and install Jaeger export pipeline
	_, flush, err := jaeger.NewExportPipeline(
		jaeger.WithCollectorEndpoint("http://192.168.162.128:14268/api/traces"),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: "tests",
			Tags: []core.KeyValue{
				key.String("exporter", "jaeger"),
			},
		}),
		jaeger.RegisterAsGlobal(),
		jaeger.WithSDK(&sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
	)
	if err != nil {
		log.Fatal(err)
	}

	return flush
}

func GetAccountBalance(seed string) float64 {

	key,_ := keypair.ParseFull(seed)
	acc, _ := GetAccount(key.Address())

	strBalance,_ := acc.GetNativeBalance()

	floatBalance,_ := strconv.ParseFloat(strBalance,64)

	return floatBalance
}

func CreateAndFundAccount(seed string) {

	client := horizonclient.DefaultTestNetClient

	pair, err := keypair.ParseFull(seed)

	if err != nil {
		log.Fatal(err)
	}

	_, errAccount := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: pair.Address()})

	if errAccount != nil {
		txSuccess, errCreate := client.Fund(pair.Address())

		if errCreate != nil {
			log.Fatal(err)
		}

		log.Printf("Account "+seed+" created - trans#:", txSuccess.Hash)
	}
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
