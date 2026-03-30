import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import DatePicker from 'react-datepicker';
import 'react-datepicker/dist/react-datepicker.css';

interface Transaction {
    date: string;
    account_name: string;
    institution_name: string;
    account_number: string;
    amount: number;
    description: string;
    category: string;
    ignored: boolean;
    hash: string;
}

const TransactionsPage: React.FC = () => {
    const [transactions, setTransactions] = useState<Transaction[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Filters
    const [accountFilter, setAccountFilter] = useState('');
    const [dateFrom, setDateFrom] = useState<Date | null>(null);
    const [dateTo, setDateTo] = useState<Date | null>(null);

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
                const [name, number] = accountFilter.split(' - ');
                if (name) params.append('account_name', name.trim());
                if (number) params.append('account_number', number.trim());
            }

            const formatDate = (d: Date) => {
                const year = d.getFullYear();
                const month = String(d.getMonth() + 1).padStart(2, '0');
                const day = String(d.getDate()).padStart(2, '0');
                return `${year}-${month}-${day}`;
            };

            if (dateFrom) params.append('date_from', formatDate(dateFrom));
            if (dateTo) params.append('date_to', formatDate(dateTo));

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
        <div className="min-h-screen bg-gray-50 p-4 md:p-8 font-sans">
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
                    <div className="p-4 md:p-5 bg-gray-50 border-b border-gray-100 grid grid-cols-1 sm:grid-cols-2 md:flex md:flex-wrap gap-3 items-end">
                        <div className="sm:col-span-2 md:flex-1 md:min-w-[200px]">
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
                        
                        <div className="min-w-0 md:flex-1 md:min-w-[150px]">
                            <label className="block text-sm font-medium text-gray-700 mb-1">From Date</label>
                            <DatePicker
                                selected={dateFrom}
                                onChange={(date: Date | null) => setDateFrom(date)}
                                className="w-full min-w-0 bg-white border border-gray-300 rounded-lg py-2 px-3 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-shadow outline-none"
                                wrapperClassName="w-full"
                                placeholderText="Select Date"
                            />
                        </div>

                        <div className="min-w-0 md:flex-1 md:min-w-[150px]">
                            <label className="block text-sm font-medium text-gray-700 mb-1">To Date</label>
                            <DatePicker
                                selected={dateTo}
                                onChange={(date: Date | null) => setDateTo(date)}
                                className="w-full min-w-0 bg-white border border-gray-300 rounded-lg py-2 px-3 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-shadow outline-none"
                                wrapperClassName="w-full"
                                placeholderText="Select Date"
                            />
                        </div>

                        <div className="sm:col-span-2 md:col-span-1">
                            <button 
                                onClick={() => { setAccountFilter(''); setDateFrom(null); setDateTo(null); }}
                                className="w-full md:w-auto py-2 px-4 bg-white border border-gray-300 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-50 transition-colors shadow-sm focus:ring-2 focus:ring-blue-500 outline-none"
                            >
                                Clear Filters
                            </button>
                        </div>
                    </div>

                    {/* Loading overlay */}
                    <div className="relative min-h-[400px]">
                        {loading && transactions.length === 0 ? (
                            <div className="absolute inset-0 flex items-center justify-center bg-white/80 z-10">
                                <span className="text-gray-500 font-medium flex items-center gap-2">
                                    <svg className="animate-spin h-5 w-5 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
                                    Loading data...
                                </span>
                            </div>
                        ) : null}

                        {/* Mobile card list */}
                        <div className="md:hidden divide-y divide-gray-100">
                            {transactions.length > 0 ? (
                                transactions.slice((currentPage - 1) * rowsPerPage, currentPage * rowsPerPage).map((t, idx) => (
                                    <div key={t.hash || idx} className="p-4 hover:bg-slate-50 transition-colors">
                                        <div className="flex items-start justify-between gap-2 mb-1">
                                            <div className="flex-1 min-w-0">
                                                <p className="text-sm font-medium text-gray-900 truncate">{t.description}</p>
                                                <p className="text-xs text-gray-500 mt-0.5">{t.account_name} &middot; x{t.account_number}</p>
                                            </div>
                                            <span className={`text-sm font-semibold whitespace-nowrap ${t.amount < 0 ? 'text-green-600' : 'text-gray-900'}`}>
                                                {t.amount < 0 ? '+' : ''}{formatCurrency(t.amount * -1)}
                                            </span>
                                        </div>
                                        <div className="flex items-center justify-between mt-2">
                                            <span className="text-xs text-gray-500">{t.date}</span>
                                            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-700 border border-gray-200">
                                                {t.category || 'Uncategorized'}
                                            </span>
                                        </div>
                                    </div>
                                ))
                            ) : (
                                !loading && (
                                    <p className="py-12 text-center text-gray-500 text-sm">
                                        No transactions found matching your criteria.
                                    </p>
                                )
                            )}
                        </div>

                        {/* Desktop table */}
                        <div className="hidden md:block overflow-x-auto">
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
                                            <tr key={t.hash || idx} className="hover:bg-slate-50 transition-colors group">
                                                <td className="py-4 px-6 text-sm text-gray-600 whitespace-nowrap">{t.date}</td>
                                                <td className="py-4 px-6 text-sm">
                                                    <div className="font-medium text-gray-900">{t.account_name}</div>
                                                    <div className="text-xs text-gray-500">x{t.account_number}</div>
                                                </td>
                                                <td className="py-4 px-6 text-sm text-gray-800 max-w-md truncate" title={t.description}>
                                                    {t.description}
                                                </td>
                                                <td className="py-4 px-6 text-sm">
                                                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 border border-gray-200">
                                                        {t.category || 'Uncategorized'}
                                                    </span>
                                                </td>
                                                <td className="py-4 px-6 text-sm text-right whitespace-nowrap">
                                                    <span className={`font-semibold ${t.amount < 0 ? 'text-green-600' : 'text-gray-900'}`}>
                                                        {t.amount < 0 ? '+' : ''}{formatCurrency(t.amount * -1)}
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
