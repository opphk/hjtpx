import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  PieChart,
  Pie,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer
} from 'recharts';

const ChartComponent = ({ type = 'line', data = [], xKey = 'name', yKeys = [], title, height = 300 }) => {
  const { t } = useTranslation();
  const [chartData, setChartData] = useState(data);

  useEffect(() => {
    setChartData(data);
  }, [data]);

  const colors = ['#8884d8', '#82ca9d', '#ffc658', '#ff7300', '#00C49F', '#FFBB28', '#FF8042'];

  const renderChart = () => {
    switch (type) {
      case 'line':
        return (
          <LineChart data={chartData} height={height}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey={xKey} />
            <YAxis />
            <Tooltip />
            <Legend />
            {yKeys.map((key, index) => (
              <Line
                key={key}
                type="monotone"
                dataKey={key}
                stroke={colors[index % colors.length]}
                strokeWidth={2}
              />
            ))}
          </LineChart>
        );

      case 'bar':
        return (
          <BarChart data={chartData} height={height}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey={xKey} />
            <YAxis />
            <Tooltip />
            <Legend />
            {yKeys.map((key, index) => (
              <Bar
                key={key}
                dataKey={key}
                fill={colors[index % colors.length]}
              />
            ))}
          </BarChart>
        );

      case 'pie':
        return (
          <PieChart height={height}>
            <Pie
              data={chartData}
              dataKey={yKeys[0] || 'value'}
              nameKey={xKey}
              cx="50%"
              cy="50%"
              outerRadius={80}
              label
            >
              {chartData.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={colors[index % colors.length]} />
              ))}
            </Pie>
            <Tooltip />
            <Legend />
          </PieChart>
        );

      case 'area':
        return (
          <AreaChart data={chartData} height={height}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey={xKey} />
            <YAxis />
            <Tooltip />
            <Legend />
            {yKeys.map((key, index) => (
              <Area
                key={key}
                type="monotone"
                dataKey={key}
                stroke={colors[index % colors.length]}
                fillOpacity={0.6}
                fill={colors[index % colors.length]}
              />
            ))}
          </AreaChart>
        );

      default:
        return null;
    }
  };

  return (
    <div className="chart-component">
      {title && <h3 className="chart-title">{title}</h3>}
      <ResponsiveContainer width="100%" height={height}>
        {renderChart()}
      </ResponsiveContainer>
    </div>
  );
};

export default ChartComponent;
