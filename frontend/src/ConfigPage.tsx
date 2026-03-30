import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';

interface CardConfig {
    ID?: number;
    Name: string;
    ImportName?: string;
    AccountNumber: string;
    Limit: number;
    StatementDay: number;
    DueDay: number;
    StartingBalance: number;
    StartingDate: string;
    StatementGraceDays: number;
}

interface AppConfig {
    Subscribers: string[];
    AlertDaysBeforeDue: number;
    Cards: CardConfig[];
    SMTP: {
        Host: string;
        Port: number;
        User: string;
        Password?: string;
    };
    Timezone: string;
}

const ConfigPage: React.FC = () => {
    const [config, setConfig] = useState<AppConfig | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [isSaving, setIsSaving] = useState(false);

    // Card editing state
    const [editingCard, setEditingCard] = useState<CardConfig | null>(null);
    const [isCardModalOpen, setIsCardModalOpen] = useState(false);

    const fetchConfig = async () => {
        try {
            setLoading(true);
            const response = await fetch('/api/config');
            if (!response.ok) {
                throw new Error('Failed to fetch configuration');
            }
            const data = await response.json();
            setConfig(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Unknown error');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchConfig();
    }, []);

    const handleSaveGlobalConfig = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!config) return;

        try {
            setIsSaving(true);
            const response = await fetch('/api/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(config),
            });

            if (!response.ok) {
                const errText = await response.text();
                throw new Error(`Save failed: ${errText}`);
            }
            alert('Configuration saved successfully!');
        } catch (err) {
            alert(err instanceof Error ? err.message : 'Save failed');
        } finally {
            setIsSaving(false);
        }
    };

    const handleDeleteCard = async (card: CardConfig) => {
        if (card.ID === undefined) return;
        if (!window.confirm(`Are you sure you want to delete ${card.Name}?`)) {
            return;
        }

        try {
            const response = await fetch(`/api/cards/${card.ID}`, {
                method: 'DELETE',
            });

            if (!response.ok) {
                const errText = await response.text();
                throw new Error(`Delete failed: ${errText}`);
            }

            // Refresh config
            await fetchConfig();
        } catch (err) {
            alert(err instanceof Error ? err.message : 'Delete failed');
        }
    };

    const handleSaveCard = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!editingCard) return;

        try {
            const response = await fetch('/api/cards', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(editingCard),
            });

            if (!response.ok) {
                const errText = await response.text();
                throw new Error(`Save card failed: ${errText}`);
            }

            setIsCardModalOpen(false);
            setEditingCard(null);
            await fetchConfig();
        } catch (err) {
            alert(err instanceof Error ? err.message : 'Save card failed');
        }
    };

    const openAddCard = () => {
        setEditingCard({
            ID: 0,
            Name: '',
            ImportName: '',
            AccountNumber: '',
            Limit: 0,
            StatementDay: 1,
            DueDay: 1,
            StartingBalance: 0,
            StartingDate: new Date().toISOString().split('T')[0],
            StatementGraceDays: 0
        });
        setIsCardModalOpen(true);
    };

    const openEditCard = (card: CardConfig) => {
        setEditingCard({ ...card });
        setIsCardModalOpen(true);
    };

    if (loading && !config) {
        return <div className="min-h-screen flex items-center justify-center text-gray-500">Loading...</div>;
    }

    return (
        <div className="min-h-screen bg-gray-50 p-4 md:p-8 font-sans">
            <div className="max-w-4xl mx-auto">
                <header className="flex flex-col md:flex-row md:items-center justify-between mb-8 gap-4">
                    <div>
                        <div className="flex items-center gap-3 mb-2">
                            <Link to="/" className="text-blue-600 hover:text-blue-800 transition-colors flex items-center gap-1 text-sm font-medium bg-blue-50 py-1 px-2 rounded-md hover:bg-blue-100">
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path></svg>
                                Back to Dashboard
                            </Link>
                        </div>
                        <h1 className="text-3xl font-bold text-gray-900 tracking-tight">Configuration</h1>
                        <p className="text-gray-500 mt-1">Manage application settings and cards</p>
                    </div>
                </header>

                {error && (
                    <div className="bg-red-50 text-red-700 p-4 rounded-lg mb-8 border border-red-100 flex items-start gap-3">
                        <svg className="w-5 h-5 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                        {error}
                    </div>
                )}

                {config && (
                    <div className="space-y-8">
                        {/* Global Settings */}
                        <section className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
                            <div className="p-6 border-b border-gray-100">
                                <h2 className="text-xl font-semibold text-gray-900">Global Settings</h2>
                            </div>
                            <form onSubmit={handleSaveGlobalConfig} className="p-6 space-y-6">
                                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                                    <div>
                                        <label className="block text-sm font-medium text-gray-700 mb-1">Timezone</label>
                                        <input 
                                            type="text" 
                                            className="w-full bg-gray-50 border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 focus:bg-white outline-none transition-all"
                                            value={config.Timezone}
                                            onChange={(e) => setConfig({ ...config, Timezone: e.target.value })}
                                            placeholder="e.America/Chicago"
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-700 mb-1">Alert Days Before Due</label>
                                        <input 
                                            type="number" 
                                            className="w-full bg-gray-50 border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 focus:bg-white outline-none transition-all"
                                            value={config.AlertDaysBeforeDue}
                                            onChange={(e) => setConfig({ ...config, AlertDaysBeforeDue: parseInt(e.target.value) || 0 })}
                                        />
                                    </div>
                                </div>

                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Subscribers (one per line)</label>
                                    <textarea 
                                        className="w-full bg-gray-50 border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 focus:bg-white outline-none transition-all h-24"
                                        value={config.Subscribers.join('\n')}
                                        onChange={(e) => setConfig({ ...config, Subscribers: e.target.value.split('\n').map(s => s.trim()).filter(s => s !== '') })}
                                        placeholder="email@example.com"
                                    />
                                </div>

                                <div className="pt-4 border-t border-gray-100 flex justify-end">
                                    <button 
                                        type="submit" 
                                        disabled={isSaving}
                                        className="bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-6 rounded-lg shadow-sm transition-colors disabled:opacity-50"
                                    >
                                        {isSaving ? 'Saving...' : 'Save Global Settings'}
                                    </button>
                                </div>
                            </form>
                        </section>

                        {/* Cards Section */}
                        <section className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
                            <div className="p-6 border-b border-gray-100 flex justify-between items-center">
                                <h2 className="text-xl font-semibold text-gray-900">Cards</h2>
                                <button 
                                    onClick={openAddCard}
                                    className="bg-green-600 hover:bg-green-700 text-white font-medium py-1.5 px-4 rounded-lg text-sm shadow-sm transition-colors flex items-center gap-1"
                                >
                                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 4v16m8-8H4"></path></svg>
                                    Add Card
                                </button>
                            </div>
                            <div className="overflow-x-auto">
                                <table className="w-full text-left border-collapse">
                                    <thead>
                                        <tr className="bg-gray-50 border-b border-gray-200">
                                            <th className="py-3 px-6 text-xs font-semibold text-gray-500 uppercase tracking-wider">Name</th>
                                            <th className="py-3 px-6 text-xs font-semibold text-gray-500 uppercase tracking-wider">Account #</th>
                                            <th className="py-3 px-6 text-xs font-semibold text-gray-500 uppercase tracking-wider text-right">Limit</th>
                                            <th className="py-3 px-6 text-xs font-semibold text-gray-500 uppercase tracking-wider text-center">Stmt/Due Day</th>
                                            <th className="py-3 px-6 text-xs font-semibold text-gray-500 uppercase tracking-wider text-right">Actions</th>
                                        </tr>
                                    </thead>
                                    <tbody className="divide-y divide-gray-100">
                                        {config.Cards.map((card) => (
                                            <tr key={card.ID} className="hover:bg-gray-50 transition-colors">
                                                <td className="py-4 px-6 text-sm font-medium text-gray-900">{card.Name}</td>
                                                <td className="py-4 px-6 text-sm text-gray-600">x{card.AccountNumber}</td>
                                                <td className="py-4 px-6 text-sm text-gray-900 text-right font-mono">${card.Limit.toLocaleString()}</td>
                                                <td className="py-4 px-6 text-sm text-center text-gray-600">{card.StatementDay} / {card.DueDay}</td>
                                                <td className="py-4 px-6 text-sm text-right">
                                                    <div className="flex justify-end gap-2">
                                                        <button 
                                                            onClick={() => openEditCard(card)}
                                                            className="p-1.5 text-blue-600 hover:bg-blue-50 rounded-md transition-colors"
                                                            title="Edit"
                                                        >
                                                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path></svg>
                                                        </button>
                                                        <button 
                                                            onClick={() => handleDeleteCard(card)}
                                                            className="p-1.5 text-red-600 hover:bg-red-50 rounded-md transition-colors"
                                                            title="Delete"
                                                        >
                                                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                                                        </button>
                                                    </div>
                                                </td>
                                            </tr>
                                        ))}
                                        {config.Cards.length === 0 && (
                                            <tr>
                                                <td colSpan={5} className="py-8 text-center text-gray-500 italic">No cards configured.</td>
                                            </tr>
                                        )}
                                    </tbody>
                                </table>
                            </div>
                        </section>

                        {/* SMTP Settings */}
                        <section className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
                            <div className="p-6 border-b border-gray-100">
                                <h2 className="text-xl font-semibold text-gray-900">Email (SMTP) Settings</h2>
                            </div>
                            <div className="p-6 grid grid-cols-1 md:grid-cols-2 gap-6">
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Host</label>
                                    <input 
                                        type="text" 
                                        className="w-full bg-gray-50 border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 focus:bg-white outline-none transition-all"
                                        value={config.SMTP.Host}
                                        onChange={(e) => setConfig({ ...config, SMTP: { ...config.SMTP, Host: e.target.value } })}
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Port</label>
                                    <input 
                                        type="number" 
                                        className="w-full bg-gray-50 border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 focus:bg-white outline-none transition-all"
                                        value={config.SMTP.Port}
                                        onChange={(e) => setConfig({ ...config, SMTP: { ...config.SMTP, Port: parseInt(e.target.value) || 0 } })}
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">User</label>
                                    <input 
                                        type="text" 
                                        className="w-full bg-gray-50 border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 focus:bg-white outline-none transition-all"
                                        value={config.SMTP.User}
                                        onChange={(e) => setConfig({ ...config, SMTP: { ...config.SMTP, User: e.target.value } })}
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Password</label>
                                    <input 
                                        type="password" 
                                        className="w-full bg-gray-50 border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 focus:bg-white outline-none transition-all"
                                        value={config.SMTP.Password || ''}
                                        onChange={(e) => setConfig({ ...config, SMTP: { ...config.SMTP, Password: e.target.value } })}
                                        placeholder="••••••••"
                                    />
                                </div>
                            </div>
                        </section>
                    </div>
                )}
            </div>

            {/* Card Modal */}
            {isCardModalOpen && editingCard && (
                <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50 animate-in fade-in duration-200">
                    <div className="bg-white rounded-xl shadow-xl w-full max-w-lg overflow-hidden animate-in zoom-in-95 duration-200">
                        <div className="p-6 border-b border-gray-100 flex justify-between items-center bg-gray-50">
                            <h3 className="text-lg font-bold text-gray-900">{editingCard.ID ? 'Edit Card' : 'Add New Card'}</h3>
                            <button onClick={() => setIsCardModalOpen(false)} className="text-gray-400 hover:text-gray-600">
                                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12"></path></svg>
                            </button>
                        </div>
                        <form onSubmit={handleSaveCard} className="p-6 space-y-4">
                            <div className="grid grid-cols-2 gap-4">
                                <div className="col-span-2">
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Card Name</label>
                                    <input 
                                        required
                                        type="text" 
                                        className="w-full border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 outline-none"
                                        value={editingCard.Name}
                                        onChange={(e) => setEditingCard({ ...editingCard, Name: e.target.value })}
                                        placeholder="e.g. Chase Sapphire"
                                    />
                                </div>
                                <div className="col-span-2">
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Import Name (Optional)</label>
                                    <input 
                                        type="text" 
                                        className="w-full border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 outline-none"
                                        value={editingCard.ImportName || ''}
                                        onChange={(e) => setEditingCard({ ...editingCard, ImportName: e.target.value })}
                                        placeholder="e.g. Discover (Used to match CSV transactions)"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Last 4 Digits</label>
                                    <input 
                                        required
                                        type="text" 
                                        maxLength={4}
                                        className="w-full border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 outline-none"
                                        value={editingCard.AccountNumber}
                                        onChange={(e) => setEditingCard({ ...editingCard, AccountNumber: e.target.value })}
                                        placeholder="1234"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Credit Limit</label>
                                    <input 
                                        required
                                        type="number" 
                                        className="w-full border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 outline-none"
                                        value={editingCard.Limit}
                                        onChange={(e) => setEditingCard({ ...editingCard, Limit: parseInt(e.target.value) || 0 })}
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Statement Day (1-31)</label>
                                    <input 
                                        required
                                        type="number" 
                                        min={1} max={31}
                                        className="w-full border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 outline-none"
                                        value={editingCard.StatementDay}
                                        onChange={(e) => setEditingCard({ ...editingCard, StatementDay: parseInt(e.target.value) || 1 })}
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Due Day (1-31)</label>
                                    <input 
                                        required
                                        type="number" 
                                        min={1} max={31}
                                        className="w-full border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 outline-none"
                                        value={editingCard.DueDay}
                                        onChange={(e) => setEditingCard({ ...editingCard, DueDay: parseInt(e.target.value) || 1 })}
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Starting Balance</label>
                                    <input 
                                        required
                                        type="number" 
                                        className="w-full border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 outline-none"
                                        value={editingCard.StartingBalance}
                                        onChange={(e) => setEditingCard({ ...editingCard, StartingBalance: parseFloat(e.target.value) || 0 })}
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Starting Date</label>
                                    <input 
                                        required
                                        type="date" 
                                        className="w-full border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 outline-none"
                                        value={editingCard.StartingDate}
                                        onChange={(e) => setEditingCard({ ...editingCard, StartingDate: e.target.value })}
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Grace Days</label>
                                    <input 
                                        required
                                        type="number" 
                                        min={0}
                                        className="w-full border border-gray-300 rounded-lg py-2 px-3 focus:ring-2 focus:ring-blue-500 outline-none"
                                        value={editingCard.StatementGraceDays}
                                        onChange={(e) => setEditingCard({ ...editingCard, StatementGraceDays: parseInt(e.target.value) || 0 })}
                                    />
                                </div>
                            </div>
                            <div className="pt-6 flex justify-end gap-3">
                                <button 
                                    type="button" 
                                    onClick={() => setIsCardModalOpen(false)}
                                    className="px-4 py-2 border border-gray-300 rounded-lg text-gray-700 font-medium hover:bg-gray-50"
                                >
                                    Cancel
                                </button>
                                <button 
                                    type="submit" 
                                    className="px-4 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 shadow-sm"
                                >
                                    Save Card
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}
        </div>
    );
};

export default ConfigPage;
