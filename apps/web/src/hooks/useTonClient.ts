import { getHttpEndpoint, getHttpV4Endpoint } from "@orbs-network/ton-access";
import { CHAIN } from "@tonconnect/ui-react";
import { TonClient, TonClient4 } from "@ton/ton";
import { useAsyncInitialize } from "./useAsyncInitialize";
import { useTonConnect } from "./useTonConnect";

export function useTonClient() {
    const { network } = useTonConnect()

    return {
        client: useAsyncInitialize(async () => {
            if (!network) return;

            if (network === CHAIN.MAINNET) {
                return new TonClient({
                    endpoint: await getHttpEndpoint({ network: "mainnet" })
                })
            } else
                return new TonClient4({
                    endpoint: await getHttpV4Endpoint({ network: "testnet" })
                })
        }, [network])
    }
}