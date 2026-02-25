import { TonClient } from "@ton/ton";
import { useAsyncInitialize } from "./useAsyncInitialize";

export function useTonClient() {
    return {
        client: useAsyncInitialize(async () => {
            // TonClient is used only like wrapper for oppening contracts. 
            // All transactions are sent via TonConnect, so endpoint and api-key is not needed.
            return new TonClient({ endpoint: "" });
        }, [])
    }
}