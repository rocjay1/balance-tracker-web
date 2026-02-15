import React from 'react';

interface CardProps {
    card_name: string;
    account_number: string;
    statement_balance: number;
    current_balance: number;
    projected_balance: number;
    target_balance: number;
    payment_needed: number;
    due_date: string;
}

const Card: React.FC<CardProps> = ({
    card_name,
    account_number,
    statement_balance,
    current_balance,
    projected_balance,
    target_balance,
    payment_needed,
    due_date,
}) => {
    const isPaid = payment_needed === 0;
    const statusColor = isPaid ? 'bg-green-500' : 'bg-red-500';
    const borderColor = isPaid ? 'border-green-200' : 'border-red-200';

    return (
        <div className={`p-6 rounded-xl shadow-lg border ${borderColor} bg-white transition-all hover:shadow-xl`}>
            <div className="flex justify-between items-start mb-4">
                <div>
                    <h3 className="text-xl font-bold text-gray-800">{card_name}</h3>
                    <p className="text-sm text-gray-500">x{account_number}</p>
                </div>
                <div className={`h-3 w-3 rounded-full ${statusColor}`} title={isPaid ? "Target Met" : "Payment Needed"}></div>
            </div>

            <div className="space-y-3">
                <div className="flex justify-between items-center text-sm">
                    <span className="text-gray-500">Due Date</span>
                    <span className="font-semibold text-gray-700">{due_date}</span>
                </div>

                <div className="pt-2 border-t border-gray-100">
                    <div className="flex justify-between items-center text-sm mb-1">
                        <span className="text-gray-500">Statement Bal</span>
                        <span className="font-medium">${statement_balance.toFixed(2)}</span>
                    </div>
                    <div className="flex justify-between items-center text-sm mb-1">
                        <span className="text-gray-500">Current Bal</span>
                        <span className="font-medium">${current_balance.toFixed(2)}</span>
                    </div>
                    <div className="flex justify-between items-center text-sm">
                        <span className="text-gray-500">Proj. Bal</span>
                        <span className="font-medium">${projected_balance.toFixed(2)}</span>
                    </div>
                </div>

                <div className="pt-3 border-t border-gray-100">
                    <div className="flex justify-between items-center text-sm mb-1">
                        <span className="text-gray-500">Target (10%)</span>
                        <span className="font-medium text-blue-600">${target_balance.toFixed(2)}</span>
                    </div>
                    <div className="flex justify-between items-center mt-2">
                        <span className="font-bold text-gray-700">Pay Extra</span>
                        <span className={`text-xl font-bold ${isPaid ? 'text-green-600' : 'text-red-500'}`}>
                            ${payment_needed.toFixed(2)}
                        </span>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default Card;
