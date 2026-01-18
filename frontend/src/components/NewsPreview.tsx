import { Calendar, RefreshCw, User } from 'lucide-react';
import React, { useState, useEffect } from 'react';

interface NewsItem {
    title: string;
    excerpt: string;
    url: string;
    date: string;
    author: string;
    imageUrl?: string;
}

interface NewsPreviewProps {
    getNews: () => Promise<NewsItem[]>
}

export const NewsPreview: React.FC<NewsPreviewProps> = ({ getNews }) => {
    const [news, setNews] = useState<NewsItem[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const fetchNews = async () => {
        setLoading(true);
        setError(null);
        try {
            const items = await getNews();
            setNews(items);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to fetch news');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchNews()
    }, []);

    const openLink = (url: string) => {
        window.open(url, '_blank');
    };

    return (
        <div className='flex flex-col w-max mx-auto gap-y-4'>
            <div className='flex justify-between'>
                <h1 className='text-white text-xl font-bold'>
                    Latest news from Hytale
                </h1>
                <button
                    onClick={fetchNews}
                    disabled={loading}
                    className="rounded-l hover:text-white disabled:opacity-50">
                    <RefreshCw size={20} className={loading ? 'animate-spin' : ''} />
                </button>
            </div>

            {loading ?
                (<div className="h-full flex items-center justify-center">
                    <div className="text-center">
                        <RefreshCw size={40} className="text-[#FFA845] animate-spin mx-auto mb-4" />
                        <p className="text-white">Loading news...</p>
                    </div>
                </div>)
                : error ? (
                    <div className="h-full flex items-center justify-center">
                        <div className="text-center">
                            <p className="text-red-400 mb-4">{error}</p>
                            <button
                                onClick={fetchNews}
                                className="px-4 py-2 bg-white/10 hover:bg-white/20 rounded-lg transition-colors"
                            >
                                Try Again
                            </button>
                        </div>
                    </div>
                ) :
                    (<div className='flex flex-col items-start gap-y-2 glass p-6 rounded-xl'>
                        {news.map((item) => {
                            return (
                                <div className='flex flex-col'>
                                    <button onClick={() => openLink(item.url)} disabled={loading} className='text-[#FFA845] hover:underline cursor-pointer'>
                                        {item.title}
                                    </button>
                                    <div className='flex flex-col text-sm'>
                                        <div className='flex gap-x-1'>
                                            <User size='16' className='opacity-50' />
                                            <p className='text-white/70'>
                                                {item.author}
                                            </p>
                                        </div>
                                        <div className='flex gap-x-1'>
                                            <Calendar size='16' className='opacity-50' />
                                            <p className='text-white/70'>
                                                {item.date}
                                            </p>
                                        </div>
                                    </div>
                                </div>

                            )
                        })}
                        <button onClick={() => openLink("https://hytale.com/news")} className='w-full font-bold hover:underline cursor-pointer'>
                            Read more
                        </button>
                    </div>)
            }

        </div>
    );
};
