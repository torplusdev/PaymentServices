#!/bin/bash

cd /opt/torplus/PaymentServices/PaymentGateway
source stellar_seed.conf
./payment-gateway $stellar_seed 28080
