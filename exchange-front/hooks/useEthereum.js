import { ethers } from "ethers";
import { useEffect, useState } from "react";

const useEthereum = () => {
    const [address, setAddress] = useState();
    const [balance, setBalance] = useState(0);
    const [signer, setSigner] = useState();

    const getBalance = async () => {
        const balance = await signer.getBalance();
        console.log("getting balance:", balance);
        setBalance(balance)
    }

    const connect = async() => {
        if (window.ethereum) {
            const provider = new ethers.providers.Web3Provider(
                window.ethereum,
                "any"
            );
            await provider.send("eth_requestAccounts", []);
            const signer = provider.getSigner();
            setSigner(signer)

            const address = await signer.getAddress();
            setAddress(address);
        }
    }
    return {
        getBalance, balance, connect, address
    };
};

export default useEthereum