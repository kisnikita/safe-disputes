import { TonClient } from "@ton/ton";
import { useAsyncInitialize } from "./useAsyncInitialize";

const TONCENTER_MAINNET_RPC = "https://toncenter.com/api/v2/jsonRPC";
const TONCENTER_TESTNET_RPC = "https://testnet.toncenter.com/api/v2/jsonRPC";

const resolveTonEndpoint = (): string => {
    const envEndpoint = import.meta.env.VITE_TONAPI_RPC_ENDPOINT as string | undefined;
    if (envEndpoint && envEndpoint.trim().length > 0) {
        return envEndpoint.trim();
    }

    const network = (import.meta.env.VITE_TON_NETWORK as string | undefined)?.toLowerCase();
    return network === "mainnet" ? TONCENTER_MAINNET_RPC : TONCENTER_TESTNET_RPC;
};

export function useTonClient() {
    return {
        client: useAsyncInitialize(async () => {
            return new TonClient({ endpoint: resolveTonEndpoint() });
        }, [])
    }
}
