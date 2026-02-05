import React, { useMemo } from 'react';
import { Tooltip } from 'antd';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { useThemeStore } from '../stores/themeStore';

interface HeatmapDataPoint {
  date: string;
  count: number;
  additions: number;
  deletions: number;
  week_day: number;
  week_of_year: number;
}

interface ContributionHeatmapProps {
  data: HeatmapDataPoint[];
  totalCount: number;
  maxCount: number;
  loading?: boolean;
}

const CELL_SIZE = 11;
const CELL_GAP = 3;
const MONTH_LABEL_HEIGHT = 20;
const WEEK_LABEL_WIDTH = 30;

const ContributionHeatmap: React.FC<ContributionHeatmapProps> = ({ 
  data, 
  totalCount, 
  maxCount,
  loading 
}) => {
  const { t } = useTranslation();
  const { isDark } = useThemeStore();

  const getColor = (count: number): string => {
    if (count === 0) return isDark ? '#161b22' : '#ebedf0';
    const intensity = Math.min(count / Math.max(maxCount, 1), 1);
    if (isDark) {
      if (intensity <= 0.25) return '#0e4429';
      if (intensity <= 0.5) return '#006d32';
      if (intensity <= 0.75) return '#26a641';
      return '#39d353';
    } else {
      if (intensity <= 0.25) return '#9be9a8';
      if (intensity <= 0.5) return '#40c463';
      if (intensity <= 0.75) return '#30a14e';
      return '#216e39';
    }
  };

  const { weeks, months } = useMemo(() => {
    if (!data || data.length === 0) return { weeks: [], months: [] };

    const weeksMap = new Map<number, HeatmapDataPoint[]>();
    const monthsSet = new Map<string, number>();

    let currentWeek = -1;
    let weekIndex = 0;
    
    data.forEach((point) => {
      const date = dayjs(point.date);
      const weekOfYear = date.isoWeek();
      
      if (weekOfYear !== currentWeek) {
        currentWeek = weekOfYear;
        weekIndex++;
        weeksMap.set(weekIndex, []);
        
        if (date.date() <= 7) {
          monthsSet.set(date.format('MMM'), weekIndex);
        }
      }
      
      const weekData = weeksMap.get(weekIndex) || [];
      weekData.push(point);
      weeksMap.set(weekIndex, weekData);
    });

    const weeksArray = Array.from(weeksMap.entries()).map(([weekIdx, days]) => ({
      weekIndex: weekIdx,
      days,
    }));

    const monthsArray = Array.from(monthsSet.entries()).map(([month, weekIdx]) => ({
      month,
      weekIndex: weekIdx,
    }));

    return { weeks: weeksArray, months: monthsArray };
  }, [data]);

  const weekDays = ['', 'Mon', '', 'Wed', '', 'Fri', ''];

  if (loading) {
    return (
      <div style={{ 
        height: 150, 
        display: 'flex', 
        alignItems: 'center', 
        justifyContent: 'center',
        color: isDark ? '#8b949e' : '#57606a'
      }}>
        {t('common.loading')}
      </div>
    );
  }

  const svgWidth = WEEK_LABEL_WIDTH + weeks.length * (CELL_SIZE + CELL_GAP) + 20;
  const svgHeight = MONTH_LABEL_HEIGHT + 7 * (CELL_SIZE + CELL_GAP) + 30;

  return (
    <div style={{ overflowX: 'auto' }}>
      <svg width={svgWidth} height={svgHeight}>
        {months.map(({ month, weekIndex }) => (
          <text
            key={month}
            x={WEEK_LABEL_WIDTH + (weekIndex - 1) * (CELL_SIZE + CELL_GAP)}
            y={12}
            fontSize={10}
            fill={isDark ? '#8b949e' : '#57606a'}
          >
            {month}
          </text>
        ))}

        {weekDays.map((day, index) => (
          <text
            key={index}
            x={0}
            y={MONTH_LABEL_HEIGHT + index * (CELL_SIZE + CELL_GAP) + CELL_SIZE - 2}
            fontSize={9}
            fill={isDark ? '#8b949e' : '#57606a'}
          >
            {day}
          </text>
        ))}

        {weeks.map(({ weekIndex, days }) => (
          <g key={weekIndex} transform={`translate(${WEEK_LABEL_WIDTH + (weekIndex - 1) * (CELL_SIZE + CELL_GAP)}, ${MONTH_LABEL_HEIGHT})`}>
            {days.map((point) => {
              const dayOfWeek = dayjs(point.date).day();
              return (
                <Tooltip
                  key={point.date}
                  title={
                    <div>
                      <div style={{ fontWeight: 'bold' }}>{dayjs(point.date).format('YYYY-MM-DD')}</div>
                      <div>{t('memberAnalysis.commitCount')}: {point.count}</div>
                      <div style={{ color: '#52c41a' }}>+{point.additions}</div>
                      <div style={{ color: '#ff4d4f' }}>-{point.deletions}</div>
                    </div>
                  }
                >
                  <rect
                    x={0}
                    y={dayOfWeek * (CELL_SIZE + CELL_GAP)}
                    width={CELL_SIZE}
                    height={CELL_SIZE}
                    rx={2}
                    fill={getColor(point.count)}
                    style={{ cursor: 'pointer' }}
                  />
                </Tooltip>
              );
            })}
          </g>
        ))}

        <g transform={`translate(${svgWidth - 120}, ${svgHeight - 18})`}>
          <text x={0} y={10} fontSize={10} fill={isDark ? '#8b949e' : '#57606a'}>
            {t('memberAnalysis.less', 'Less')}
          </text>
          {[0, 0.25, 0.5, 0.75, 1].map((intensity, i) => (
            <rect
              key={i}
              x={30 + i * (CELL_SIZE + 2)}
              y={0}
              width={CELL_SIZE}
              height={CELL_SIZE}
              rx={2}
              fill={getColor(intensity * maxCount)}
            />
          ))}
          <text x={100} y={10} fontSize={10} fill={isDark ? '#8b949e' : '#57606a'}>
            {t('memberAnalysis.more', 'More')}
          </text>
        </g>
      </svg>

      <div style={{ 
        marginTop: 8, 
        fontSize: 12, 
        color: isDark ? '#8b949e' : '#57606a' 
      }}>
        {t('memberAnalysis.contributionsInYear', '{{count}} contributions in the last year', { count: totalCount })}
      </div>
    </div>
  );
};

export default ContributionHeatmap;
