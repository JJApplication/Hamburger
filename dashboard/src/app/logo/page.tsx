import React from 'react';
import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';

export default function LogoPage() {
  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-black overflow-hidden [perspective:1000px] relative">
      
      {/* Back to Home Button */}
      <Link 
        href="/" 
        className="absolute top-8 left-8 z-50 flex items-center gap-2 px-4 py-2 text-sm font-medium text-white/80 bg-white/10 rounded-full backdrop-blur-md border border-white/20 transition-all hover:bg-white/20 hover:text-white shadow-lg"
      >
        <ArrowLeft size={16} />
        返回主页
      </Link>

      <div className="relative flex items-center justify-center w-[300px] h-[300px] animate-spin-3d [transform-style:preserve-3d]">
        
        {/* Bottom Bun */}
        <div className="absolute inset-0 m-auto w-[260px] h-[260px] bg-orange-400 rounded-full border-[8px] border-orange-500 shadow-[inset_-20px_-20px_40px_rgba(0,0,0,0.4),0_15px_30px_rgba(0,0,0,0.6)] [transform:translateZ(-90px)]"></div>

        {/* Beef 2 */}
        <div className="absolute inset-0 m-auto w-[270px] h-[270px] bg-[#4a2e15] rounded-full border-[12px] border-[#3a2005] shadow-[inset_-15px_-15px_30px_rgba(0,0,0,0.6),0_10px_20px_rgba(0,0,0,0.5)] [transform:translateZ(-40px)]">
          {/* Grill marks/texture */}
          <div className="absolute top-16 left-32 w-5 h-5 bg-[#3a2005] rounded-full opacity-50"></div>
          <div className="absolute bottom-20 right-16 w-4 h-4 bg-[#3a2005] rounded-full opacity-50"></div>
          <div className="absolute top-32 right-32 w-6 h-6 bg-[#3a2005] rounded-full opacity-50"></div>
        </div>

        {/* Cheese 2 */}
        <div className="absolute inset-0 m-auto w-[192px] h-[192px] bg-yellow-400 opacity-90 rounded-sm border-[4px] border-yellow-500 shadow-[0_5px_10px_rgba(0,0,0,0.4)] [transform:translateZ(-10px)_rotate(-12deg)]"></div>

        {/* Beef 1 */}
        <div className="absolute inset-0 m-auto w-[270px] h-[270px] bg-[#4a2e15] rounded-full border-[12px] border-[#3a2005] shadow-[inset_-15px_-15px_30px_rgba(0,0,0,0.6),0_10px_20px_rgba(0,0,0,0.5)] [transform:translateZ(30px)]">
          {/* Grill marks/texture */}
          <div className="absolute top-10 left-10 w-4 h-4 bg-[#3a2005] rounded-full opacity-50"></div>
          <div className="absolute top-20 right-20 w-6 h-6 bg-[#3a2005] rounded-full opacity-50"></div>
          <div className="absolute bottom-16 left-32 w-5 h-5 bg-[#3a2005] rounded-full opacity-50"></div>
        </div>

        {/* Cheese 1 */}
        <div className="absolute inset-0 m-auto w-[192px] h-[192px] bg-yellow-400 opacity-90 rounded-sm border-[4px] border-yellow-500 shadow-[0_5px_10px_rgba(0,0,0,0.4)] [transform:translateZ(60px)_rotate(45deg)]"></div>

        {/* Lettuce */}
        <div className="absolute inset-0 m-auto w-[280px] h-[280px] bg-green-500 rounded-full opacity-95 border-[8px] border-green-600 shadow-[inset_-10px_-10px_20px_rgba(0,0,0,0.3),0_10px_15px_rgba(0,0,0,0.5)] [transform:translateZ(80px)]">
          {/* Lettuce frills */}
          <div className="absolute -top-2 -left-2 w-16 h-16 bg-green-500 rounded-full border-4 border-green-600"></div>
          <div className="absolute top-10 -right-4 w-20 h-20 bg-green-500 rounded-full border-4 border-green-600"></div>
          <div className="absolute -bottom-4 left-20 w-24 h-24 bg-green-500 rounded-full border-4 border-green-600"></div>
        </div>

        {/* Top Bun */}
        <div className="absolute inset-0 m-auto w-[260px] h-[260px] bg-orange-400 rounded-full overflow-hidden shadow-[inset_-20px_-20px_40px_rgba(0,0,0,0.4),0_10px_20px_rgba(0,0,0,0.5)] border-[8px] border-orange-500 [transform:translateZ(120px)]">
          {/* Sesame seeds */}
          <div className="absolute top-10 left-20 w-3 h-5 bg-orange-200 rounded-full rotate-45 shadow-sm"></div>
          <div className="absolute top-16 left-40 w-3 h-5 bg-orange-200 rounded-full -rotate-12 shadow-sm"></div>
          <div className="absolute top-20 right-24 w-3 h-5 bg-orange-200 rounded-full rotate-12 shadow-sm"></div>
          <div className="absolute top-32 right-12 w-3 h-5 bg-orange-200 rounded-full -rotate-45 shadow-sm"></div>
          <div className="absolute top-40 left-16 w-3 h-5 bg-orange-200 rounded-full rotate-90 shadow-sm"></div>
          <div className="absolute top-48 left-32 w-3 h-5 bg-orange-200 rounded-full rotate-45 shadow-sm"></div>
          <div className="absolute top-56 right-24 w-3 h-5 bg-orange-200 rounded-full -rotate-12 shadow-sm"></div>
          <div className="absolute top-24 left-28 w-3 h-5 bg-orange-200 rounded-full rotate-12 shadow-sm"></div>
          <div className="absolute top-36 left-44 w-3 h-5 bg-orange-200 rounded-full -rotate-12 shadow-sm"></div>
          <div className="absolute top-44 right-32 w-3 h-5 bg-orange-200 rounded-full rotate-45 shadow-sm"></div>
        </div>

      </div>
    </div>
  );
}
