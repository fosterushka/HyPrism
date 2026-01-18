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
        <div className='flex flex-col gap-y-4 max-w-md'>
            <div className='flex justify-between items-center'>
                <h2 className='text-white text-lg font-bold'>
                    Latest Hytale News
                </h2>
                <button
                    onClick={fetchNews}
                    disabled={loading}
                    className="rounded-lg hover:text-white disabled:opacity-50 transition-colors">
                    <RefreshCw size={18} className={loading ? 'animate-spin' : ''} />
                </button>
            </div>

            {loading ?
                (<div className="flex items-center justify-center py-8">
                    <div className="text-center">
                        <RefreshCw size={32} className="text-[#FFA845] animate-spin mx-auto mb-3" />
                        <p className="text-white/70 text-sm">Loading news...</p>
                    </div>
                </div>)
                : error ? (
                    <div className="flex items-center justify-center py-8">
                        <div className="text-center">
                            <p className="text-red-400 mb-3 text-sm">{error}</p>
                            <button
                                onClick={fetchNews}
                                className="px-4 py-2 bg-white/10 hover:bg-white/20 rounded-lg transition-colors text-sm"
                            >
                                Try Again
                            </button>
                        </div>
                    </div>
                ) :
                    (<div className='flex flex-col gap-y-3 glass p-4 rounded-xl'>
                        {news.map((item, index) => {
                            return (
                                <div key={index} className='flex gap-3 group'>
                                    {/* News Image */}
                                    {item.imageUrl && (
                                        <img 
                                            src={item.imageUrl} 
                                            alt={item.title}
                                            className='w-24 h-16 object-cover rounded-lg flex-shrink-0'
                                        />
                                    )}
                                    {/* News Content */}
                                    <div className='flex flex-col justify-center min-w-0'>
                                        <button 
                                            onClick={() => openLink(item.url)} 
                                            disabled={loading} 
                                            className='text-[#FFA845] hover:underline cursor-pointer text-left text-sm font-medium line-clamp-2 mb-1'>
                                            {item.title}
                                        </button>
                                        <div className='flex flex-col text-xs'>
                                            <div className='flex gap-x-1 items-center text-white/60'>
                                                <User size='12' />
                                                <p className='truncate'>{item.author}</p>
                                            </div>
                                            <div className='flex gap-x-1 items-center text-white/60'>
                                                <Calendar size='12' />
                                                <p>{item.date}</p>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            )
                        })}
                        <button 
                            onClick={() => openLink("https://hytale.com/news")} 
                            className='w-full font-semibold hover:underline cursor-pointer text-[#FFA845] text-sm mt-1'>
                            Read more on hytale.com →
                        </button>
                    </div>)
            }

        </div>
    );
};
