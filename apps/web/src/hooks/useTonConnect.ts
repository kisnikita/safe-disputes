import { useTonConnectUI, useTonWallet } from "@tonconnect/ui-react";
import { SenderArguments, Sender, Address } from "@ton/core";

export function useTonConnect(): {
    sender: Sender;
    address: string | null;
    connected: boolean;
} {
    const [tonConnectUI] = useTonConnectUI()
    const wallet = useTonWallet()

    return {
        sender: {
            send: async (args: SenderArguments) => {
                await tonConnectUI.sendTransaction(
                    {
                        messages: [
                            {
                                address: args.to.toString(),
                                amount: args.value.toString(),
                                payload: args.body?.toBoc().toString("base64"),
                            },
                        ],
                        validUntil: Date.now() + 5 * 60 * 1000, // 5 minutes for user to approve
                    },
                    {
                        notifications: []
                    }
                );
            },
            address: wallet?.account?.address ? Address.parse(wallet?.account?.address) : undefined
        },
        address: wallet?.account.address ?? null,
        connected: !!wallet,
    }
}
