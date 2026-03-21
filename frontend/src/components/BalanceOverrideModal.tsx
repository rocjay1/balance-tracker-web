import React, { useState, useEffect } from 'react';

interface BalanceOverrideModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSuccess: () => void;
    cardName: string;
    accountNumber: string;
    balanceValue: number;
    hasOverride: boolean;
}

const BalanceOverrideModal: React.FC<BalanceOverrideModalProps> = ({
    isOpen,
    onClose,
    onSuccess,
    cardName,
    accountNumber,
    balanceValue,
    hasOverride,
}) => {
    const [balance, setBalance] = useState<string>(balanceValue.toString());
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (isOpen) {
            setBalance(balanceValue.toString());
            setError(null);
        }
    }, [isOpen, balanceValue]);

    if (!isOpen) return null;

    const handleSave = async () => {
        const val = parseFloat(balance);
        if (isNaN(val)) {
            setError('Please enter a valid number');
            return;
        }

        setLoading(true);
        setError(null);

        try {
            const res = await fetch(`/api/overrides/${accountNumber}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ statement_balance: val }),
            });

            if (!res.ok) {
                const text = await res.text();
                throw new Error(text || 'Failed to save override');
            }

            onSuccess();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'An error occurred');
        } finally {
            setLoading(false);
        }
    };

    const handleRemove = async () => {
        setLoading(true);
        setError(null);

        try {
            const res = await fetch(`/api/overrides/${accountNumber}`, {
                method: 'DELETE',
            });

            if (!res.ok) {
                const text = await res.text();
                throw new Error(text || 'Failed to remove override');
            }

            onSuccess();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'An error occurred');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 font-sans p-4">
            <div className="bg-white rounded-xl shadow-2xl w-full max-w-md overflow-hidden">
                <div className="px-6 py-4 border-b border-gray-100 flex justify-between items-center">
                    <h3 className="text-lg font-bold text-gray-800">Edit Statement Balance</h3>
                    <button onClick={onClose} className="text-gray-400 hover:text-gray-600 transition-colors">
                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12"></path></svg>
                    </button>
                </div>
                
                <div className="p-6 space-y-4">
                    <div>
                        <p className="text-sm text-gray-500 mb-1">Card</p>
                        <p className="font-medium text-gray-800">{cardName} (x{accountNumber})</p>
                    </div>

                    <div>
                        <label className="block text-sm text-gray-500 mb-1" htmlFor="balanceInput">
                            Statement Balance Override
                        </label>
                        <div className="relative">
                            <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 font-medium">$</span>
                            <input
                                id="balanceInput"
                                type="number"
                                step="0.01"
                                className="w-full pl-7 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all"
                                value={balance}
                                onChange={(e) => setBalance(e.target.value)}
                                placeholder="0.00"
                                disabled={loading}
                            />
                        </div>
                            This will override the calculated statement balance for the current billing cycle.
                    </div>

                    {error && (
                        <div className="p-3 bg-red-50 text-red-600 rounded-lg border border-red-100 text-sm">
                            {error}
                        </div>
                    )}
                </div>

                <div className="px-6 py-4 bg-gray-50 border-t border-gray-100 flex justify-between items-center">
                    {hasOverride ? (
                        <button
                            onClick={handleRemove}
                            disabled={loading}
                            className="text-red-500 hover:text-red-700 hover:bg-red-50 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors disabled:opacity-50"
                        >
                            Remove Override
                        </button>
                    ) : (
                        <div></div>
                    )}
                    <div className="flex gap-2">
                        <button
                            onClick={onClose}
                            disabled={loading}
                            className="px-4 py-2 text-gray-600 hover:bg-gray-100 rounded-lg font-medium transition-colors disabled:opacity-50"
                        >
                            Cancel
                        </button>
                        <button
                            onClick={handleSave}
                            disabled={loading}
                            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors shadow-sm disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                        >
                            {loading ? (
                                <>
                                    <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-white" fill="none" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
                                    Saving...
                                </>
                            ) : (
                                "Save Override"
                            )}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default BalanceOverrideModal;
