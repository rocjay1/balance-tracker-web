import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';

interface Transaction {
    Date: string;
    AccountName: string;
    InstitutionName: string;
    AccountNumber: string;
    Amount: number;
    Description: string;
    Category: string;
    Ignored: boolean;
    Hash: string;
}

const TransactionsPage: React.FC = () => {
    const [transactions, setTransactions] = useState<Transaction[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Filters
    const [accountFilter, setAccountFilter] = useState('');
    const [dateFrom, setDateFrom] = useState('');
    const [dateTo, setDateTo] = useState('');

    // Pagination
    const [rowsPerPage, setRowsPerPage] = useState(50);
    const [currentPage, setCurrentPage] = useState(1);
    
    // Dynamic lists for filters
    const [availableAccounts, setAvailableAccounts] = useState<{name: string, number: string}[]>([]);

    const fetchTransactions = async () => {
        try {
            setLoading(true);
            const params = new URLSearchParams();
            if (accountFilter) {
                // simple split assuming format 'Name - Last4'
                const [name, number] = accountFilter.split(' - ');
                if (name) params.append('account_name', name.trim());
                if (number) params.append('account_number', number.trim());
            }
            if (dateFrom) params.append('date_from', dateFrom);
            if (dateTo) params.append('date_to', dateTo);

            const response = await fetch(`/api/transactions?${params.toString()}`);
            if (!response.ok) {
                throw new Error('Failed to fetch transactions');
            }
            const data = await response.json();
            setTransactions(data || []);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Unknown error');
        } finally {
            setLoading(false);
        }
    };

    const fetchAccounts = async () => {
        try {
            const response = await fetch('/api/status');
            if (!response.ok) {
                throw new Error('Failed to fetch accounts');
            }
            const data: unknown = await response.json();
            if (Array.isArray(data)) {
                const accounts = data.map(card => {
                    const c = card as Record<string, unknown>;
                    return {
                        name: typeof c.card_name === 'string' ? c.card_name : 'Unknown',
                        number: typeof c.account_number === 'string' ? c.account_number : ''
                    };
                });
                setAvailableAccounts(accounts);
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Unknown error');
        }
    };

    useEffect(() => {
        fetchAccounts();
    }, []);

    useEffect(() => {
        fetchTransactions();
        setCurrentPage(1);
    }, [accountFilter, dateFrom, dateTo]);

    const formatCurrency = (amount: number) => {
        return new Intl.NumberFormat('en-US', {
            style: 'currency',
            currency: 'USD',
        }).format(amount);
    };

    return (
        <div className="min-h-screen bg-gray-50 p-8 font-sans">
            <div className="max-w-7xl mx-auto">
                <header className="flex flex-col md:flex-row md:items-center justify-between mb-8 gap-4">
                    <div>
                        <div className="flex items-center gap-3 mb-2">
                            <Link to="/" className="text-blue-600 hover:text-blue-800 transition-colors flex items-center gap-1 text-sm font-medium bg-blue-50 py-1 px-2 rounded-md hover:bg-blue-100">
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path></svg>
                                Back to Dashboard
                            </Link>
                        </div>
                        <h1 className="text-3xl font-bold text-gray-900 tracking-tight">Transactions</h1>
                        <p className="text-gray-500 mt-1">View and filter your transaction history</p>
                    </div>
                </header>

                {error && (
                    <div className="bg-red-50 text-red-700 p-4 rounded-lg mb-8 border border-red-100 flex items-start gap-3">
                        <svg className="w-5 h-5 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                        {error}
                    </div>
                )}

                <div className="bg-white rounded-xl shadow-lg border border-gray-100 overflow-hidden mb-8">
                    {/* Filters Bar */}
                    <div className="p-5 bg-gray-50 border-b border-gray-100 flex flex-wrap gap-4 items-start">
                        <div className="flex-1 min-w-[200px]">
                            <label className="block text-sm font-medium text-gray-700 mb-1">Account</label>
                            <select 
                                className="w-full bg-white border border-gray-300 rounded-lg py-2 px-3 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-shadow outline-none"
                                value={accountFilter}
                                onChange={(e) => setAccountFilter(e.target.value)}
                            >
                                <option value="">All Accounts</option>
                                {availableAccounts.map(acc => (
                                    <option key={`${acc.name}-${acc.number}`} value={`${acc.name} - ${acc.number}`}>
                                        {acc.name} (x{acc.number})
                                    </option>
                                ))}
                            </select>
                        </div>
                        
                        <div className="flex-1 min-w-[150px]">
                            <label className="block text-sm font-medium text-gray-700 mb-1">From Date</label>
                            <input 
                                type="date" 
                                className="w-full bg-white border border-gray-300 rounded-lg py-2 px-3 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-shadow outline-none"
                                value={dateFrom}
                                onChange={(e) => setDateFrom(e.target.value)}
                            />
                        </div>

                        <div className="flex-1 min-w-[150px]">
                            <label className="block text-sm font-medium text-gray-700 mb-1">To Date</label>
                            <input 
                                type="date" 
                                className="w-full bg-white border border-gray-300 rounded-lg py-2 px-3 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-shadow outline-none"
                                value={dateTo}
                                onChange={(e) => setDateTo(e.target.value)}
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1 invisible">Clear</label>
                            <button 
                                onClick={() => { setAccountFilter(''); setDateFrom(''); setDateTo(''); }}
                                className="py-2 px-4 bg-white border border-gray-300 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-50 transition-colors shadow-sm focus:ring-2 focus:ring-blue-500 outline-none"
                            >
                                Clear Filters
                            </button>
                        </div>
                    </div>

                    {/* Table */}
                    <div className="overflow-x-auto relative min-h-[400px]">
                        {loading && transactions.length === 0 ? (
                            <div className="absolute inset-0 flex items-center justify-center bg-white/80 z-10">
                                <span className="text-gray-500 font-medium flex items-center gap-2">
                                    <svg className="animate-spin h-5 w-5 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
                                    Loading data...
                                </span>
                            </div>
                        ) : null}

                        <table className="w-full text-left border-collapse">
                            <thead>
                                <tr className="bg-white border-b border-gray-200">
                                    <th className="py-3 px-6 text-xs font-semibold text-gray-500 uppercase tracking-wider">Date</th>
                                    <th className="py-3 px-6 text-xs font-semibold text-gray-500 uppercase tracking-wider">Account</th>
                                    <th className="py-3 px-6 text-xs font-semibold text-gray-500 uppercase tracking-wider">Description</th>
                                    <th className="py-3 px-6 text-xs font-semibold text-gray-500 uppercase tracking-wider">Category</th>
                                    <th className="py-3 px-6 text-xs font-semibold text-gray-500 uppercase tracking-wider text-right">Amount</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-gray-100">
                                {transactions.length > 0 ? (
                                    transactions.slice((currentPage - 1) * rowsPerPage, currentPage * rowsPerPage).map((t, idx) => (
                                        <tr key={t.Hash || idx} className="hover:bg-slate-50 transition-colors group">
                                            <td className="py-4 px-6 text-sm text-gray-600 whitespace-nowrap">{t.Date}</td>
                                            <td className="py-4 px-6 text-sm">
                                                <div className="font-medium text-gray-900">{t.AccountName}</div>
                                                <div className="text-xs text-gray-500">x{t.AccountNumber}</div>
                                            </td>
                                            <td className="py-4 px-6 text-sm text-gray-800 max-w-md truncate" title={t.Description}>
                                                {t.Description}
                                            </td>
                                            <td className="py-4 px-6 text-sm">
                                                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 border border-gray-200">
                                                    {t.Category || 'Uncategorized'}
                                                </span>
                                            </td>
                                            <td className="py-4 px-6 text-sm text-right whitespace-nowrap">
                                                <span className={`font-semibold ${t.Amount < 0 ? 'text-green-600' : 'text-gray-900'}`}>
                                                    {t.Amount < 0 ? '+' : ''}{formatCurrency(t.Amount * -1)}
                                                </span>
                                            </td>
                                        </tr>
                                    ))
                                ) : (
                                    !loading && (
                                        <tr>
                                            <td colSpan={5} className="py-12 text-center text-gray-500">
                                                No transactions found matching your criteria.
                                            </td>
                                        </tr>
                                    )
                                )}
                            </tbody>
                        </table>
                    </div>

                    {/* Pagination Footer */}
                    {transactions.length > 0 && (
                        <div className="px-5 py-3 bg-gray-50 border-t border-gray-100 flex flex-wrap items-center justify-between gap-3 text-sm text-gray-600">
                            <div className="flex items-center gap-2">
                                <span>Rows per page:</span>
                                <select
                                    className="bg-white border border-gray-300 rounded-md py-1 px-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                                    value={rowsPerPage}
                                    onChange={(e) => { setRowsPerPage(Number(e.target.value)); setCurrentPage(1); }}
                                >
                                    <option value={50}>50</option>
                                    <option value={100}>100</option>
                                    <option value={200}>200</option>
                                </select>
                            </div>
                            <div className="flex items-center gap-3">
                                <span>
                                    {Math.min((currentPage - 1) * rowsPerPage + 1, transactions.length)}&ndash;{Math.min(currentPage * rowsPerPage, transactions.length)} of {transactions.length}
                                </span>
                                <button
                                    onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                                    disabled={currentPage === 1}
                                    className="py-1 px-3 bg-white border border-gray-300 rounded-md text-sm font-medium hover:bg-gray-50 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                                >
                                    Previous
                                </button>
                                <button
                                    onClick={() => setCurrentPage(p => Math.min(Math.ceil(transactions.length / rowsPerPage), p + 1))}
                                    disabled={currentPage >= Math.ceil(transactions.length / rowsPerPage)}
                                    className="py-1 px-3 bg-white border border-gray-300 rounded-md text-sm font-medium hover:bg-gray-50 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                                >
                                    Next
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default TransactionsPage;
