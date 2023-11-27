import useEthereum from "@/hooks/useEthereum";
import { useEffect } from "react";
import { ethers } from "ethers";

const Spacer = () => {
  return <div className="py-2"></div>;
};

const PlaceOrderCard = () => {
  return (
    <Card title="Place market order">
      <div>
        <div>Place market order</div>
        <Spacer />
        <div>
          <input className="py-2 px-4 text-gray-900 text-sm bg-white rounded-xl appearance-none outline-none" />
        </div>
      </div>
    </Card>
  );
};

const Card = ( {children, title} ) => {
  return (
    <div className="p-6 bg-teal-500 rounded-xl text-white">
      <h1 className="text-gray-800 font-bold text-xl mb-2">{title}</h1>
      {children}
    </div>
  );
};

export default function Home() {
  const { connect, getBalance, balance, address } = useEthereum();

  //useEffect(() => {}, [])

  useEffect(() => {
    connect();
  }, []);

  useEffect(() => {
      if (address) {
        getBalance();
      }
  }, [address]);

  return (
    <div>
      <Navigation />
      <div className="container mx-auto">
        <div className="flex justify-between mb-20">
          <div className="p-6 bg-teal-500 rounded-xl text-xl font-semibold ">
            balance: {ethers.utils.formatUnits( balance.toString() )}
          </div>
        </div>
        <div className="flex space-x-10">
          <Card title="Orderbook">xx</Card>
          <PlaceOrderCard></PlaceOrderCard>
        </div>
      </div>
    </div>
  );
};


const Button =({ children, onClick }) => {
  return (
    <button 
      onClick={onClick} 
      className="p-3 bg-blue-500 font-bold text-white"
    >
      {children}
    </button>
  );
};


const Navigation = () => {
  const { connect, address } = useEthereum();

  return (
    <div className="container mx-auto py-8 mb-20">
      <div className="flex justify-between">
        <div>
          <a>ExchangeGG</a>
        </div>
        <div className="flex space-x-6 align-middle">
          <a>portfolio</a>
          <a>help</a>
          {address ? (
            <div>{address}</div> 
          ) : (
            <Button onClick={connect}>Connect</Button>
          )}
          
        </div>
      </div>
    </div>
  );
};
