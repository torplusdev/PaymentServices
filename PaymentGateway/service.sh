#!/bin/bash

cd /opt/paidpiper/PaymentServices/PaymentGateway
source stellar_seed.conf
./payment-gateway $stellar_seed 28080
