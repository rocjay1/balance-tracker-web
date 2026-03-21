import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import Card from './components/Card';
import BalanceOverrideModal from './components/BalanceOverrideModal';

interface CardData {
    card_name: string;
    account_number: string;
    statement_balance: number;
    current_balance: number;
    projected_balance: number;
    target_balance: number;
    payment_needed: number;
    due_date: string;
    has_override: boolean;
}

const Dashboard: React.FC = () => {
    const [cards, setCards] = useState<CardData[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [selectedCard, setSelectedCard] = useState<CardData | null>(null);

    const fetchCards = async () => {
        try {
            const response = await fetch('/api/status');
            if (!response.ok) {
                throw new Error('Failed to fetch data');
            }
            const data = await response.json();
            setCards(data || []);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Unknown error');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchCards();
    }, []);

    const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
        if (!e.target.files || e.target.files.length === 0) return;

        const file = e.target.files[0];
        const formData = new FormData();
        formData.append('file', file);

        try {
            setLoading(true);
            const response = await fetch('/api/upload', {
                method: 'POST',
                body: formData,
            });

            if (!response.ok) {
                const errText = await response.text();
                throw new Error(`Upload failed: ${errText}`);
            }

            // Refresh data
            await fetchCards();
            alert('Upload successful!');
        } catch (err) {
            alert(err instanceof Error ? err.message : 'Upload failed');
            setLoading(false);
        }
    };

    if (loading && cards.length === 0) {
        return <div className="min-h-screen flex items-center justify-center text-gray-500">Loading...</div>;
    }

    return (
        <div className="min-h-screen bg-gray-50 p-8 font-sans">
            <div className="max-w-7xl mx-auto">
                <header className="flex flex-col md:flex-row md:items-center justify-between mb-10 gap-4">
                    <div>
                        <h1 className="text-3xl font-bold text-gray-900 tracking-tight">Balance Tracker</h1>
                        <p className="text-gray-500 mt-1">Optimize your credit utilization</p>
                    </div>

                    <div className="flex items-center gap-4">
                        <Link to="/transactions" className="p-2 text-gray-500 hover:text-blue-600 transition-colors font-medium flex items-center gap-2 border border-transparent hover:bg-blue-50 hover:border-blue-100 rounded-lg px-3">
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg>
                            Transactions
                        </Link>
                        <label className="cursor-pointer bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded-lg shadow-sm transition-colors flex items-center gap-2">
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"></path></svg>
                            <span>Import</span>
                            <input type="file" className="hidden" accept=".csv" onChange={handleFileUpload} />
                        </label>
                        <button onClick={fetchCards} className="p-2 text-gray-400 hover:text-gray-600 transition-colors" title="Refresh">
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
                        </button>
                    </div>
                </header>

                {error && (
                    <div className="bg-red-50 text-red-700 p-4 rounded-lg mb-8 border border-red-100 flex items-start gap-3">
                        <svg className="w-5 h-5 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                        {error}
                    </div>
                )}

                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                    {cards.map((card, idx) => (
                        <Card 
                            key={`${card.card_name}-${card.account_number}-${idx}`} 
                            {...card} 
                            onEditBalance={() => setSelectedCard(card)}
                        />
                    ))}
                </div>

                {cards.length === 0 && !loading && !error && (
                    <div className="text-center py-20 text-gray-400">
                        No cards found. Check your configuration.
                    </div>
                )}
            </div>

            <BalanceOverrideModal
                isOpen={selectedCard !== null}
                onClose={() => setSelectedCard(null)}
                onSuccess={() => {
                    setSelectedCard(null);
                    fetchCards();
                }}
                cardName={selectedCard?.card_name || ''}
                accountNumber={selectedCard?.account_number || ''}
                currentBalance={selectedCard?.statement_balance || 0}
                hasOverride={selectedCard?.has_override || false}
            />
        </div>
    );
};

export default Dashboard;
