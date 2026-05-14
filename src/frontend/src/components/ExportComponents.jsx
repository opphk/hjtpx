import React from 'react';
import Papa from 'papaparse';
import { format } from 'date-fns';
import { useTranslation } from 'react-i18next';

const ExportButton = ({ data, filename, format = 'csv', fields, onExport }) => {
  const { t } = useTranslation();

  const handleExport = () => {
    let exportData = data;
    let fileContent;
    let mimeType;
    let extension;

    switch (format) {
      case 'csv':
        extension = 'csv';
        mimeType = 'text/csv';
        fileContent = Papa.unparse(exportData, { fields });
        break;

      case 'json':
        extension = 'json';
        mimeType = 'application/json';
        fileContent = JSON.stringify(exportData, null, 2);
        break;

      case 'xlsx':
        handleExcelExport(exportData, filename);
        return;

      case 'pdf':
        handlePdfExport(exportData, filename);
        return;

      default:
        return;
    }

    downloadFile(fileContent, `${filename}.${extension}`, mimeType);

    if (onExport) {
      onExport({ success: true, format, filename: `${filename}.${extension}` });
    }
  };

  const handleExcelExport = async (data, filename) => {
    try {
      const ExcelJS = await import('exceljs');
      const workbook = new ExcelJS.Workbook();
      const worksheet = workbook.addWorksheet('Export');

      if (data.length > 0) {
        const headers = fields || Object.keys(data[0]);
        worksheet.addRow(headers);

        data.forEach(item => {
          const row = headers.map(header => item[header]);
          worksheet.addRow(row);
        });
      }

      const buffer = await workbook.xlsx.writeBuffer();
      downloadFile(buffer, `${filename}.xlsx`, 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet');

      if (onExport) {
        onExport({ success: true, format: 'xlsx', filename: `${filename}.xlsx` });
      }
    } catch (error) {
      console.error('Excel export error:', error);
      if (onExport) {
        onExport({ success: false, error: error.message });
      }
    }
  };

  const handlePdfExport = async (data, filename) => {
    try {
      const PDFDocument = (await import('pdfkit')).default;
      const doc = new PDFDocument();
      const chunks = [];

      doc.on('data', chunk => chunks.push(chunk));
      doc.on('end', () => {
        const pdfBuffer = Buffer.concat(chunks);
        downloadFile(pdfBuffer, `${filename}.pdf`, 'application/pdf');
      });

      doc.fontSize(20).text('Export Report', { align: 'center' });
      doc.moveDown();
      doc.fontSize(12).text(`Generated: ${format(new Date(), 'yyyy-MM-dd HH:mm:ss')}`);
      doc.moveDown();

      if (data.length > 0) {
        const headers = fields || Object.keys(data[0]);
        doc.text(headers.join('\t'));
        doc.moveDown();

        data.forEach((item, index) => {
          const row = headers.map(header => item[header]).join('\t');
          doc.text(`${index + 1}. ${row}`);
        });
      }

      doc.end();

      if (onExport) {
        onExport({ success: true, format: 'pdf', filename: `${filename}.pdf` });
      }
    } catch (error) {
      console.error('PDF export error:', error);
      if (onExport) {
        onExport({ success: false, error: error.message });
      }
    }
  };

  const downloadFile = (content, filename, mimeType) => {
    const blob = content instanceof Buffer
      ? new Blob([content], { type: mimeType })
      : new Blob([content], { type: mimeType });

    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  return (
    <button onClick={handleExport} className="export-button">
      {t('export.download')} {format.toUpperCase()}
    </button>
  );
};

const ExportModal = ({ isOpen, onClose, data, defaultFilename }) => {
  const { t } = useTranslation();
  const [selectedFormat, setSelectedFormat] = React.useState('csv');
  const [filename, setFilename] = React.useState(defaultFilename || 'export');
  const [selectedFields, setSelectedFields] = React.useState([]);

  if (!isOpen) return null;

  const formats = [
    { value: 'csv', label: t('export.csv') },
    { value: 'json', label: t('export.json') },
    { value: 'xlsx', label: t('export.excel') },
    { value: 'pdf', label: t('export.pdf') }
  ];

  const availableFields = data.length > 0 ? Object.keys(data[0]) : [];

  return (
    <div className="export-modal-overlay" onClick={onClose}>
      <div className="export-modal" onClick={(e) => e.stopPropagation()}>
        <h2>{t('export.title')}</h2>

        <div className="form-group">
          <label>{t('export.filename')}</label>
          <input
            type="text"
            value={filename}
            onChange={(e) => setFilename(e.target.value)}
            placeholder="export"
          />
        </div>

        <div className="form-group">
          <label>{t('export.format')}</label>
          <select value={selectedFormat} onChange={(e) => setSelectedFormat(e.target.value)}>
            {formats.map(format => (
              <option key={format.value} value={format.value}>
                {format.label}
              </option>
            ))}
          </select>
        </div>

        <div className="form-group">
          <label>{t('export.selectedFields')}</label>
          <div className="fields-list">
            {availableFields.map(field => (
              <label key={field} className="field-checkbox">
                <input
                  type="checkbox"
                  checked={selectedFields.includes(field)}
                  onChange={(e) => {
                    if (e.target.checked) {
                      setSelectedFields([...selectedFields, field]);
                    } else {
                      setSelectedFields(selectedFields.filter(f => f !== field));
                    }
                  }}
                />
                {field}
              </label>
            ))}
          </div>
        </div>

        <div className="modal-actions">
          <button onClick={onClose} className="btn-cancel">
            {t('common.cancel')}
          </button>
          <ExportButton
            data={data}
            filename={filename}
            format={selectedFormat}
            fields={selectedFields.length > 0 ? selectedFields : undefined}
            onExport={(result) => {
              if (result.success) {
                onClose();
              }
            }}
          />
        </div>
      </div>
    </div>
  );
};

export { ExportButton, ExportModal };
export default ExportButton;
